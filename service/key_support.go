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
	"github.com/axzcnzxis/myapi/setting/ratio_setting"
	"gorm.io/gorm"
)

// init wires the subscription flow to the token-snapshot rewrite. The model
// package cannot import this one (service already imports model), so it exposes
// a registration hook instead of depending on the derivation policy directly.
func init() {
	model.RegisterDerivedTokenGroupsSyncer(SyncDerivedTokenGroupsTx)
}

// ResolveKeyGroups derives the ordered group set an API key should carry for the
// given user: the groups served by the user's active subscriptions plus the
// groups the wallet allow-list grants, ordered by the administrator-configured
// priority (e.g. vip before svip), falling back to the user's own base group —
// normally "default" — when neither source yields a group.
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
	return deriveKeyGroups(userGroup, subGroups, WalletGroupsForUser(userId)), nil
}

// WalletGroupsForUser returns the groups the wallet allow-list grants to the
// user, or nil when the wallet is unrestricted. The per-user override wins over
// the global policy, mirroring the billing path.
func WalletGroupsForUser(userId int) []string {
	userSetting, err := model.GetUserSetting(userId, false)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to read setting of user %d for wallet groups: %v", userId, err))
		return nil
	}
	return keysupport.WalletGroupsFor(userSetting.GetWalletBalanceGroups())
}

// WalletModelsForUser returns the models the wallet allow-list grants to the
// user, or nil when the wallet is unrestricted. The per-user override wins over
// the global policy, mirroring the billing path.
func WalletModelsForUser(userId int) []string {
	userSetting, err := model.GetUserSetting(userId, false)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to read setting of user %d for wallet models: %v", userId, err))
		return nil
	}
	return keysupport.WalletModelsFor(userSetting.GetWalletBalanceModels())
}

// WalletVisibleModels narrows the models a token advertises so the wallet path
// never lists more than it may pay for.
//
// An empty wallet model allow-list means "no restriction", so models is returned
// unchanged. Otherwise the result keeps the models served by the user's active
// subscriptions — a paid plan keeps advertising everything it covers — plus the
// allow-list members the token can actually reach, because the wallet may pay
// for exactly those. A balance-only account therefore advertises the
// administrator's list instead of its base group's whole catalogue.
//
// The caller passes the models the token could reach anyway (its derived groups
// already filtered by the token's own model limits), so a listed model that no
// reachable group serves is never advertised.
func WalletVisibleModels(userId int, models []string) []string {
	allowed := WalletModelsForUser(userId)
	if len(allowed) == 0 {
		return models
	}
	keep := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		keep[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	subGroups, err := model.GetActiveUserSubscriptionGroups(userId)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to read subscriptions of user %d for wallet models: %v", userId, err))
	} else {
		for _, name := range GetGroupsEnabledModels(subGroups) {
			keep[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
		}
	}
	filtered := make([]string, 0, len(models))
	for _, name := range models {
		if _, ok := keep[strings.ToLower(strings.TrimSpace(name))]; ok {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

// EffectiveAutoGroups resolves the groups a token bound to the "auto" group may
// route through right now. It is applied on every request, so a subscription
// that was added, invalidated, deleted or has just expired takes effect
// immediately without waiting for the stored snapshot to be rewritten.
//
// The stored snapshot is only a fallback: when the user's group or subscriptions
// cannot be read, the previously stored set is returned so an unrelated database
// hiccup never leaves the key unroutable. Otherwise the current entitlement —
// the groups served by the user's active subscriptions, plus the groups the
// wallet allow-list grants, or their base group when neither applies — wins, so
// a key created before a plan was purchased starts serving that plan's models at
// once. The result is never empty.
func EffectiveAutoGroups(userId int, snapshot []string) []string {
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

	return deriveKeyGroups(userGroup, subGroups, WalletGroupsForUser(userId))
}

// SyncDerivedTokenGroupsTx rewrites the stored Auto snapshot of every token the
// user owns that is bound to the "auto" group. The subscription flow calls it
// inside the same transaction that changed the user's entitlement, so the
// console, the router and the billing path observe the same group set right
// away. userGroup and subscriptionGroups are the values the caller just derived
// from the transaction, so no extra read can race with it.
func SyncDerivedTokenGroupsTx(tx *gorm.DB, userId int, userGroup string, subscriptionGroups []string) error {
	derived := deriveKeyGroups(userGroup, subscriptionGroups, WalletGroupsForUser(userId))
	if _, err := model.RewriteDerivedTokenGroupsTx(tx, userId, derived); err != nil {
		return err
	}
	return nil
}

// deriveKeyGroups is the single derivation policy shared by token creation/edit
// (ResolveKeyGroups) and the subscription-driven rewrite
// (SyncDerivedTokenGroupsTx), so a stored snapshot and a request-time resolution
// can never disagree.
//
// It merges two sources:
//
//   - the groups served by the user's active subscriptions, which must still be
//     selectable for that user; and
//   - the groups the wallet allow-list grants (walletGroups), which deliberately
//     bypass that entitlement check — an administrator listing a group there
//     explicitly allows the wallet to pay for it, so a balance-only account must
//     be able to route through it. A group ratio is still required so a typo
//     cannot point a key at a group that does not exist.
//
// Fusion only narrows the subscription part (with fusion off a single
// subscription group survives); the granted wallet groups are always kept. The
// merged set is ordered by the configured priority and capped at MaxGroups.
func deriveKeyGroups(userGroup string, subGroups []string, walletGroups []string) []string {
	// Order first, then narrow: with fusion off the single surviving group must
	// be the highest-priority one, not whichever the plan happened to list first.
	candidates := keysupport.OrderGroups(subGroups)
	if !keysupport.FusionEnabled() && len(candidates) > 1 {
		candidates = candidates[:1]
	}

	groups := make([]string, 0, len(candidates)+len(walletGroups)+1)
	seen := make(map[string]struct{}, len(candidates)+len(walletGroups)+1)
	appendGroup := func(group string) {
		group = strings.TrimSpace(group)
		if group == "" {
			return
		}
		if _, ok := seen[group]; ok {
			return
		}
		seen[group] = struct{}{}
		groups = append(groups, group)
	}

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
		appendGroup(group)
	}

	for _, group := range walletGroups {
		group = strings.TrimSpace(group)
		if group == "" || group == "auto" {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		if !ratio_setting.ContainsGroupRatio(group) {
			continue
		}
		appendGroup(group)
	}

	if len(groups) == 0 {
		groups = append(groups, baseUserGroup(userGroup))
	}

	// Re-order the merged set: a granted group may outrank a subscribed one.
	groups = keysupport.OrderGroups(groups)

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
