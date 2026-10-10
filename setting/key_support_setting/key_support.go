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
//  1. How an API key's group set is derived. When Enabled and
//     DeriveFromSubscription are both true, a key created with group "auto"
//     receives the groups served by the user's active subscriptions instead of
//     a hand-picked list, so one key transparently covers every subscribed
//     plan (and the wallet's base group when nothing is subscribed).
//  2. Which groups/models the wallet balance may pay for (Balance*).
type KeySupportSetting struct {
	// Enabled is the master switch; when false every behaviour below is off and
	// keys keep the previous hand-picked semantics.
	Enabled bool `json:"enabled"`
	// DeriveFromSubscription makes "auto" keys inherit the user's subscription
	// groups at creation/edit time.
	DeriveFromSubscription bool `json:"derive_from_subscription"`
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

	// BalanceEnabled turns on the wallet model/group allow-list.
	BalanceEnabled bool `json:"balance_enabled"`
	// BalanceGroups restricts the wallet to these groups (empty = no restriction).
	BalanceGroups []string `json:"balance_groups"`
	// BalanceModels restricts the wallet to these models (empty = no restriction).
	BalanceModels []string `json:"balance_models"`
}

var keySupportSetting = KeySupportSetting{
	Enabled:                false,
	DeriveFromSubscription: true,
	FusionEnabled:          true,
	ExposeGroups:           true,
	MaxGroups:              DefaultMaxGroups,
	GroupPriority:          []string{"vip", "svip"},
	BalanceEnabled:         false,
	BalanceGroups:          []string{},
	BalanceModels:          []string{},
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

// IsEnabled reports whether the key-support policy is active at all.
func IsEnabled() bool { return keySupportSetting.Enabled }

// DeriveFromSubscription reports whether "auto" keys should be derived from the
// user's active subscriptions.
func DeriveFromSubscription() bool {
	return keySupportSetting.Enabled && keySupportSetting.DeriveFromSubscription
}

// FusionEnabled reports whether several subscription groups may share one key.
func FusionEnabled() bool {
	return keySupportSetting.Enabled && keySupportSetting.FusionEnabled
}

// ExposeGroups reports whether the console should expand a key's real groups.
func ExposeGroups() bool {
	return keySupportSetting.Enabled && keySupportSetting.ExposeGroups
}

// MaxGroups returns the effective derived-group cap.
func MaxGroups() int {
	if keySupportSetting.MaxGroups > 0 {
		return keySupportSetting.MaxGroups
	}
	return DefaultMaxGroups
}

// BalanceEnabled reports whether the wallet allow-list is active.
func BalanceEnabled() bool {
	return keySupportSetting.Enabled && keySupportSetting.BalanceEnabled
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
// the given group/model. A disabled switch or an empty list means "no
// restriction", so the previous behaviour is preserved until an administrator
// opts in.
func CheckWalletAllowed(group, model string) error {
	if !BalanceEnabled() {
		return nil
	}
	if len(keySupportSetting.BalanceGroups) > 0 && !containsFold(keySupportSetting.BalanceGroups, group) {
		return fmt.Errorf("余额不可用于分组 %q，请使用对应的订阅套餐", group)
	}
	if len(keySupportSetting.BalanceModels) > 0 && !containsFold(keySupportSetting.BalanceModels, model) {
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
