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
package key_support_setting

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/axzcnzxis/myapi/setting/config"
)

// ConfigName is the hierarchical config module name. Every option below is
// persisted as "<ConfigName>.<json_field>" and exported into the option map
// automatically by config.GlobalConfig.
const ConfigName = "key_support_setting"

// DefaultMaxGroups caps how many groups a derived key may carry when the
// administrator leaves MaxGroups unset.
const DefaultMaxGroups = 5

// KeySupportSetting controls two related administrator policies:
//
//  1. How an API key's group set is derived. A key bound to the "auto" group
//     always receives the groups served by the user's active subscriptions
//     instead of a hand-picked list, so one key transparently covers every
//     subscribed plan (and the wallet's base group when nothing is subscribed).
//     FusionEnabled/GroupPriority/MaxGroups tune that derivation. There is no
//     on/off switch: the user's current subscriptions are the source of truth.
//  2. Which groups/models the wallet balance may pay for (Balance*). Both lists
//     are self-applying: a non-empty group list restricts the wallet path *and*
//     grants reachability, because a listed group is added to the derived group
//     set of an "auto" key even when the user's own entitlement does not cover
//     it, so a balance-only account can route through it; a non-empty model list
//     restricts the wallet path *and* is the only catalogue that path
//     advertises, so a balance-only account is offered exactly the listed models
//     instead of its base group's whole catalogue. Subscription groups and the
//     models they serve are never limited by these lists. An empty list means
//     "no restriction and no grant". There is no separate on/off switch — the
//     list itself is the switch.
type KeySupportSetting struct {
	// FusionEnabled allows several subscription groups to be fused into one key.
	// When false only the highest-priority subscription group is used.
	FusionEnabled bool `json:"fusion_enabled"`
	// ExposeGroups makes the console expand the actual group set of a key
	// instead of only showing "auto".
	ExposeGroups bool `json:"expose_groups"`
	// MaxGroups caps the derived group set (<=0 falls back to DefaultMaxGroups).
	MaxGroups int `json:"max_groups"`
	// GroupPriority is an ordered list; groups earlier in the list win when the
	// same model is served by several subscription groups (e.g. vip before
	// svip). Groups absent from the list keep their subscription order and are
	// placed after the listed ones.
	GroupPriority []string `json:"group_priority"`

	// BalanceGroups is the wallet group allow-list. Non-empty: the wallet may
	// only pay for these groups and the list grants them to "auto" keys. Empty:
	// no restriction and no grant.
	BalanceGroups []string `json:"balance_groups"`
	// BalanceModels is the wallet model allow-list. Non-empty: the wallet may
	// only pay for these models and the wallet path advertises exactly them, so a
	// balance-only account no longer sees its base group's whole catalogue.
	// Empty: no restriction.
	BalanceModels []string `json:"balance_models"`
}

var keySupportSetting = KeySupportSetting{
	FusionEnabled: true,
	ExposeGroups:  true,
	MaxGroups:     DefaultMaxGroups,
	GroupPriority: []string{"vip", "svip"},
	BalanceGroups: []string{},
	BalanceModels: []string{},
}

func init() {
	config.GlobalConfig.Register(ConfigName, &keySupportSetting)
}

// GetSetting returns a copy safe to hand to callers.
func GetSetting() KeySupportSetting {
	s := keySupportSetting
	s.GroupPriority = cloneStrings(keySupportSetting.GroupPriority)
	s.BalanceGroups = cloneStrings(keySupportSetting.BalanceGroups)
	s.BalanceModels = cloneStrings(keySupportSetting.BalanceModels)
	return s
}

// FusionEnabled reports whether several subscription groups may share one key.
func FusionEnabled() bool {
	return keySupportSetting.FusionEnabled
}

// ExposeGroups reports whether the console should expand a key's real groups.
func ExposeGroups() bool {
	return keySupportSetting.ExposeGroups
}

// MaxGroups returns the effective derived-group cap.
func MaxGroups() int {
	if keySupportSetting.MaxGroups > 0 {
		return keySupportSetting.MaxGroups
	}
	return DefaultMaxGroups
}

// WalletGroupsFor returns the effective wallet group allow-list for a user, or
// nil when the list is empty or holds no usable group.
//
// The list is self-applying: a non-empty result is both the set of groups the
// wallet may pay for and the set of groups the allow-list makes reachable for
// the user's "auto" keys. A nil result means the wallet is unrestricted, which
// also means it grants no extra group.
func WalletGroupsFor(overrideGroups []string) []string {
	groups := keySupportSetting.BalanceGroups
	if overrideGroups != nil {
		groups = overrideGroups
	}
	return normalizeNames(groups)
}

// WalletModelsFor returns the effective wallet model allow-list for a user, or
// nil when the list is empty or holds no usable entry.
//
// Like WalletGroupsFor the list is self-applying: a non-empty result is both the
// set of models the wallet may pay for and the set of models the wallet path
// advertises, so a balance-only account is offered exactly the listed models
// instead of its base group's whole catalogue. A nil result means the wallet is
// unrestricted, so everything the key can reach is advertised as before.
func WalletModelsFor(overrideModels []string) []string {
	models := keySupportSetting.BalanceModels
	if overrideModels != nil {
		models = overrideModels
	}
	return normalizeNames(models)
}

// normalizeNames trims every entry, drops blanks and de-duplicates
// case-insensitively, returning nil when nothing usable is left.
func normalizeNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// ContainsGroup reports whether list contains group, ignoring case and
// surrounding whitespace.
func ContainsGroup(list []string, group string) bool {
	return containsFold(list, group)
}

// OrderGroups reorders groups so that members of GroupPriority come first, in
// the configured order, and the remaining groups keep their original order.
func OrderGroups(groups []string) []string {
	if len(groups) == 0 {
		return groups
	}
	priority := keySupportSetting.GroupPriority
	rank := make(map[string]int, len(priority))
	for index, group := range priority {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if _, ok := rank[group]; !ok {
			rank[group] = index
		}
	}
	ordered := make([]string, len(groups))
	copy(ordered, groups)
	sort.SliceStable(ordered, func(i, j int) bool {
		ri, iok := rank[ordered[i]]
		rj, jok := rank[ordered[j]]
		if iok != jok {
			return iok
		}
		if !iok {
			return false
		}
		return ri < rj
	})
	return ordered
}

// CheckWalletAllowed returns an error when the wallet balance must not pay for
// the given group/model. An empty list means "no restriction", so the previous
// behaviour is preserved until an administrator lists something.
func CheckWalletAllowed(group, model string) error {
	return CheckWalletAllowedForUser(nil, nil, group, model)
}

// CheckWalletAllowedForUser is the per-user counterpart of CheckWalletAllowed.
// An administrator may override the global wallet allow-list for a single user:
// overrideGroups/overrideModels replace the global lists when non-nil; a nil
// slice inherits the global list. A non-nil but empty slice means "no
// restriction" for that dimension.
//
// A non-empty effective list is self-applying: the wallet may only pay for its
// members. An empty effective list means "no restriction".
func CheckWalletAllowedForUser(overrideGroups, overrideModels []string, group, model string) error {
	groups := WalletGroupsFor(overrideGroups)
	if len(groups) > 0 && !ContainsGroup(groups, group) {
		return fmt.Errorf("余额不可用于分组 %q，请使用对应的订阅套餐", group)
	}
	models := WalletModelsFor(overrideModels)
	if len(models) > 0 && !containsFold(models, model) {
		return fmt.Errorf("模型 %q 不在余额可用范围内", model)
	}
	return nil
}

// ValidateOption validates an administrator-supplied option value before it is
// persisted. It is invoked by the option-update controller.
func ValidateOption(key, value string) error {
	field := strings.TrimPrefix(key, ConfigName+".")
	switch field {
	case "max_groups":
		count, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || count <= 0 {
			return fmt.Errorf("最大分组数必须是正整数")
		}
	case "group_priority", "balance_groups", "balance_models":
		if _, err := parseStringList(value); err != nil {
			return fmt.Errorf("%s 必须是字符串数组: %w", field, err)
		}
	}
	return nil
}

func parseStringList(value string) ([]string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var list []string
	if err := json.Unmarshal([]byte(trimmed), &list); err != nil {
		return nil, err
	}
	return list, nil
}

func containsFold(list []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

func cloneStrings(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}
