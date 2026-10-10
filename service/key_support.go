/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package service

import (
	"fmt"
	"strings"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/model"
	keysupport "github.com/axzcnzxis/myapi/setting/key_support_setting"
	"gorm.io/gorm"
)

// init wires the subscription flow to the token-snapshot rewrite. The model
// package cannot import this one (service already imports model), so it exposes
// a registration hook instead of depending on the derivation policy directly.
func init() {
	model.RegisterDerivedTokenGroupsSyncer(SyncDerivedTokenGroupsTx)
}

// ResolveKeyGroups derives the ordered group set an API key should carry for the
// given user: the groups served by the user's active subscriptions, ordered by
// the administrator-configured priority (e.g. vip before svip), falling back to
// the user's own base group — normally "default" — when nothing is subscribed.
//
// The order matters: the Auto-group selector serves a request from the first
// group that has a channel for the requested model, so an earlier group wins
// whenever several of them serve the same model.
func ResolveKeyGroups(userId int) ([]string, error) {
	userGroup, err := model.GetUserGroup(userId, false)
	if err != nil {
		return nil, err
	}

	subGroups, err := model.GetActiveUserSubscriptionGroups(userId)
	if err != nil {
		return nil, err
	}
	return deriveKeyGroupsFromSubscriptions(userGroup, subGroups), nil
}

// EffectiveAutoGroups narrows the stored Auto snapshot of a token to the groups
// the user's *current* active subscriptions still serve. It is applied on every
// request for a token whose group is "auto" while subscription derivation is on,
// so a subscription that was invalidated, deleted or has just expired stops
// routing (and billing) immediately, without waiting for the stored snapshot to
// be rewritten.
//
// The snapshot may only narrow the entitlement: when the intersection is empty —
// a fully stale snapshot — the entitlement set wins instead, because an empty
// Auto set would leave the key unroutable. A user without any active
// subscription falls back to their base group. The result is never empty.
func EffectiveAutoGroups(userId int, snapshot []string) []string {
	if !keysupport.DeriveFromSubscription() {
		return snapshot
	}
	userGroup, err := model.GetUserGroup(userId, false)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to read group of user %d for auto groups: %v", userId, err))
		return snapshot
	}
	subGroups, err := model.GetActiveUserSubscriptionGroups(userId)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to read subscriptions of user %d for auto groups: %v", userId, err))
		return snapshot
	}

	entitled := deriveKeyGroupsFromSubscriptions(userGroup, subGroups)
	if len(snapshot) == 0 {
		return entitled
	}
	if narrowed := intersectGroups(entitled, snapshot); len(narrowed) > 0 {
		return narrowed
	}
	return entitled
}

// SyncDerivedTokenGroupsTx rewrites the stored Auto snapshot of every token the
// user owns that is bound to the "auto" group. The subscription flow calls it
// inside the same transaction that changed the user's entitlement, so the
// console, the router and the billing path observe the same group set right
// away. userGroup and subscriptionGroups are the values the caller just derived
// from the transaction, so no extra read can race with it.
func SyncDerivedTokenGroupsTx(tx *gorm.DB, userId int, userGroup string, subscriptionGroups []string) error {
	if !keysupport.DeriveFromSubscription() {
		return nil
	}
	derived := deriveKeyGroupsFromSubscriptions(userGroup, subscriptionGroups)
	if _, err := model.RewriteDerivedTokenGroupsTx(tx, userId, derived); err != nil {
		return err
	}
	return nil
}

// deriveKeyGroupsFromSubscriptions is the single derivation policy shared by
// token creation/edit (ResolveKeyGroups) and the subscription-driven rewrite
// (SyncDerivedTokenGroupsTx), so a stored snapshot and a request-time resolution
// can never disagree.
func deriveKeyGroupsFromSubscriptions(userGroup string, subGroups []string) []string {
	if !keysupport.FusionEnabled() && len(subGroups) > 1 {
		subGroups = subGroups[:1]
	}

	candidates := keysupport.OrderGroups(subGroups)
	groups := make([]string, 0, len(candidates)+1)
	seen := make(map[string]struct{}, len(candidates)+1)
	for _, group := range candidates {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		// Skip groups the user may not select, so a stale plan configuration can
		// never grant a token a group it is not entitled to.
		if !IsUserSelectableGroup(userGroup, group) {
			continue
		}
		seen[group] = struct{}{}
		groups = append(groups, group)
	}

	if len(groups) == 0 {
		groups = append(groups, baseUserGroup(userGroup))
	}

	if maxGroups := keysupport.MaxGroups(); maxGroups > 0 && len(groups) > maxGroups {
		groups = groups[:maxGroups]
	}
	return groups
}

// baseUserGroup returns the group a key falls back to when the user holds no
// usable subscription group: their primary group when it is selectable, and the
// default group otherwise.
func baseUserGroup(userGroup string) string {
	base := model.PrimaryUserGroup(userGroup)
	if base == "" || !IsUserSelectableGroup(userGroup, base) {
		base = model.DefaultUserGroup
	}
	return base
}

// intersectGroups keeps the members of allowed that also appear in snapshot,
// preserving allowed's order.
func intersectGroups(allowed, snapshot []string) []string {
	keep := make(map[string]struct{}, len(snapshot))
	for _, group := range snapshot {
		keep[strings.TrimSpace(group)] = struct{}{}
	}
	out := make([]string, 0, len(allowed))
	for _, group := range allowed {
		if _, ok := keep[group]; ok {
			out = append(out, group)
		}
	}
	return out
}
