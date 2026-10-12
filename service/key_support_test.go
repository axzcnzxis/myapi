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
	"path/filepath"
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/model"
	"github.com/axzcnzxis/myapi/setting"
	"github.com/axzcnzxis/myapi/setting/config"
	keysupport "github.com/axzcnzxis/myapi/setting/key_support_setting"
	"github.com/axzcnzxis/myapi/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupKeySupportTestDB gives the derivation policy a real (in-memory) database
// so it is exercised through the same model queries the request path uses.
func setupKeySupportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousRedis, previousPath := model.DB, common.RedisEnabled, common.SQLitePath

	// The group-column name shared by every model query is only initialised by
	// InitDB, so run it against a throwaway SQLite file before swapping in the
	// in-memory database this test actually queries.
	common.SQLitePath = filepath.Join(t.TempDir(), "key-support-init.db")
	require.NoError(t, model.InitDB())

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSubscription{}))
	model.DB = db
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedis
		common.SQLitePath = previousPath
	})
	return db
}

func configureKeySupportGroups(t *testing.T) {
	t.Helper()
	originalAuto := setting.AutoGroups2JsonString()
	originalUsable := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default","vip"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAuto))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})
}

// configureProductionKeySupportGroups reproduces the shipped default options
// observed on a production install: the usable groups only list default/vip
// (there is no UserUsableGroups option row, so the built-in default applies)
// while svip exists as a ratio and is reachable only through the user's own
// group membership.
func configureProductionKeySupportGroups(t *testing.T) {
	t.Helper()
	originalAuto := setting.AutoGroups2JsonString()
	originalUsable := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default","vip"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"svip":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAuto))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})
}

// configureKeySupportPolicy swaps the derivation policy (priority, fusion, cap)
// and restores it afterwards.
func configureKeySupportPolicy(t *testing.T, priority []string, fusion bool, maxGroups int) {
	t.Helper()
	policy, ok := config.GlobalConfig.Get(keysupport.ConfigName).(*keysupport.KeySupportSetting)
	require.True(t, ok, "key support setting must be registered in the global config")
	original := *policy
	policy.GroupPriority = priority
	policy.FusionEnabled = fusion
	policy.MaxGroups = maxGroups
	t.Cleanup(func() { *policy = original })
}

func seedKeySupportUser(t *testing.T, db *gorm.DB, id int, group string) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:       id,
		Username: fmt.Sprintf("key-support-%d", id),
		Password: "password",
		Group:    group,
		Status:   common.UserStatusEnabled,
	}).Error)
}

// configureKeySupportWallet swaps the wallet allow-list (groups + models) and
// restores it afterwards. It mutates the very struct registered in the global
// config, exactly like configureKeySupportPolicy does.
func configureKeySupportWallet(t *testing.T, groups, models []string) {
	t.Helper()
	policy, ok := config.GlobalConfig.Get(keysupport.ConfigName).(*keysupport.KeySupportSetting)
	require.True(t, ok, "key support setting must be registered in the global config")
	original := *policy
	policy.BalanceGroups = groups
	policy.BalanceModels = models
	t.Cleanup(func() { *policy = original })
}

// seedKeySupportUserWithSetting seeds a user carrying a per-user setting JSON,
// which is what the per-user wallet override is read from.
func seedKeySupportUserWithSetting(t *testing.T, db *gorm.DB, id int, group, setting string) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:       id,
		Username: fmt.Sprintf("key-support-%d", id),
		Password: "password",
		Group:    group,
		Status:   common.UserStatusEnabled,
		AffCode:  fmt.Sprintf("aff-%d", id),
		Setting:  setting,
	}).Error)
}

func seedKeySupportSubscription(t *testing.T, db *gorm.DB, userId int, group string) {
	t.Helper()
	now := common.GetTimestamp()
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:       userId,
		PlanId:       1,
		Status:       "active",
		Source:       "admin",
		StartTime:    now - 60,
		EndTime:      now + 3600,
		UpgradeGroup: group,
	}).Error)
}

// seedKeySupportAbility registers one served model for a group, which is what
// GetGroupsEnabledModels reads.
func seedKeySupportAbility(t *testing.T, db *gorm.DB, group, modelName string) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&model.Ability{}))
	require.NoError(t, db.Create(&model.Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: 1,
		Enabled:   true,
	}).Error)
}

// The wallet allow-list is the wallet path's whole catalogue: a balance-only
// account must be offered exactly the listed models instead of its base group's
// entire catalogue, which is the behaviour the administrator complained about.
func TestWalletVisibleModelsNarrowsToWalletAllowList(t *testing.T) {
	configureKeySupportWallet(t, nil, []string{"agnes-3.0-flash", "agnes-2.5-flash"})
	db := setupKeySupportTestDB(t)
	seedKeySupportAbility(t, db, "default", "agnes-3.0-flash")
	seedKeySupportAbility(t, db, "default", "agnes-2.5-flash")
	seedKeySupportAbility(t, db, "default", "gemini-2.5-flash")
	seedKeySupportAbility(t, db, "default", "hunyuan-hy4")
	seedKeySupportUser(t, db, 501, "default")

	reachable := GetGroupsEnabledModels([]string{"default"})
	assert.ElementsMatch(t,
		[]string{"agnes-3.0-flash", "agnes-2.5-flash", "gemini-2.5-flash", "hunyuan-hy4"},
		reachable)

	visible := WalletVisibleModels(501, reachable)
	assert.ElementsMatch(t, []string{"agnes-3.0-flash", "agnes-2.5-flash"}, visible)
}

// An empty allow-list keeps the previous behaviour: everything the key can reach
// stays advertised, so the feature is inert until an administrator lists a model.
func TestWalletVisibleModelsUnrestrictedWhenListEmpty(t *testing.T) {
	configureKeySupportWallet(t, nil, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportAbility(t, db, "default", "gemini-2.5-flash")
	seedKeySupportUser(t, db, 502, "default")

	reachable := GetGroupsEnabledModels([]string{"default"})
	assert.Equal(t, reachable, WalletVisibleModels(502, reachable))
}

// A paid plan is never limited by the wallet list: its subscription catalogue is
// always kept, and only the rest of the wallet-reachable catalogue is narrowed.
func TestWalletVisibleModelsKeepsSubscriptionCatalogue(t *testing.T) {
	configureKeySupportWallet(t, nil, []string{"agnes-3.0-flash"})
	db := setupKeySupportTestDB(t)
	seedKeySupportAbility(t, db, "vip", "vip-only-model")
	seedKeySupportAbility(t, db, "default", "agnes-3.0-flash")
	seedKeySupportAbility(t, db, "default", "default-extra-model")
	seedKeySupportUser(t, db, 503, "default")
	seedKeySupportSubscription(t, db, 503, "vip")

	reachable := []string{"agnes-3.0-flash", "default-extra-model", "vip-only-model"}
	assert.ElementsMatch(t,
		[]string{"agnes-3.0-flash", "vip-only-model"},
		WalletVisibleModels(503, reachable))
}

// A listed model that no reachable group serves is never advertised, so the
// allow-list cannot promise a model the key cannot actually route.
func TestWalletVisibleModelsHidesUnreachableAllowListEntry(t *testing.T) {
	configureKeySupportWallet(t, nil, []string{"agnes-3.0-flash", "ghost-model"})
	db := setupKeySupportTestDB(t)
	seedKeySupportAbility(t, db, "default", "agnes-3.0-flash")
	seedKeySupportUser(t, db, 504, "default")

	reachable := GetGroupsEnabledModels([]string{"default"})
	assert.Equal(t, []string{"agnes-3.0-flash"}, WalletVisibleModels(504, reachable))
}

// The per-user override replaces the global model list, exactly like the group
// list does, and an explicit empty override restores the unrestricted behaviour.
func TestWalletVisibleModelsHonoursPerUserOverride(t *testing.T) {
	configureKeySupportWallet(t, nil, []string{"agnes-3.0-flash"})
	db := setupKeySupportTestDB(t)
	seedKeySupportAbility(t, db, "default", "agnes-3.0-flash")
	seedKeySupportAbility(t, db, "default", "default-extra-model")
	seedKeySupportUserWithSetting(t, db, 505, "default", `{"wallet_balance_models":"[\"default-extra-model\"]"}`)
	seedKeySupportUserWithSetting(t, db, 506, "default", `{"wallet_balance_models":"[]"}`)

	reachable := []string{"agnes-3.0-flash", "default-extra-model"}
	assert.Equal(t, []string{"default-extra-model"}, WalletVisibleModels(505, reachable))
	assert.Equal(t, reachable, WalletVisibleModels(506, reachable))
}

// EffectiveAutoGroups is evaluated on every request, so a snapshot stored before
// a plan was bought must not keep the key limited to the base group, and a
// hand-picked snapshot must never widen the entitlement.
func TestEffectiveAutoGroupsFollowsCurrentSubscriptions(t *testing.T) {
	tests := []struct {
		name         string
		userGroup    string
		subscription string
		snapshot     []string
		expected     []string
	}{
		{name: "no subscription keeps the base group", userGroup: "default", snapshot: []string{"default"}, expected: []string{"default"}},
		{name: "a stale snapshot is replaced by the subscribed group", userGroup: "default", subscription: "vip", snapshot: []string{"default"}, expected: []string{"vip"}},
		{name: "the subscribed group survives an unchanged snapshot", userGroup: "default", subscription: "vip", snapshot: []string{"vip"}, expected: []string{"vip"}},
		{name: "a hand-picked snapshot cannot widen the entitlement", userGroup: "default", subscription: "vip", snapshot: []string{"default", "vip"}, expected: []string{"vip"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureKeySupportGroups(t)
			db := setupKeySupportTestDB(t)
			seedKeySupportUser(t, db, 201, test.userGroup)
			if test.subscription != "" {
				seedKeySupportSubscription(t, db, 201, test.subscription)
			}

			assert.Equal(t, test.expected, EffectiveAutoGroups(201, test.snapshot))
		})
	}
}

// The stored snapshot is the fallback when the entitlement cannot be read, so an
// unreadable user never leaves the key unroutable.
func TestEffectiveAutoGroupsFallsBackToSnapshotWhenEntitlementUnreadable(t *testing.T) {
	configureKeySupportGroups(t)
	setupKeySupportTestDB(t)

	assert.Equal(t, []string{"vip"}, EffectiveAutoGroups(0, []string{"vip"}))
}

// A production-shaped account: the user belongs to default,vip,svip, holds two
// active subscriptions (vip+svip and default) and the usable-group option only
// lists default/vip. Every one of the three groups must survive the derivation,
// ordered by the configured priority, so a key created for this account is never
// limited to the default group.
func TestResolveKeyGroupsMatchesSubscribedProductionAccount(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 301, "default,vip,svip")
	seedKeySupportSubscription(t, db, 301, "vip,svip")
	seedKeySupportSubscription(t, db, 301, "default")

	groups, err := ResolveKeyGroups(301)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip", "vip", "default"}, groups)
}

// With fusion disabled only the highest-priority subscribed group survives, so
// the production account collapses to svip instead of the whole subscribed set.
func TestResolveKeyGroupsKeepsOnlyTopGroupWhenFusionDisabled(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, false, 5)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 302, "default,vip,svip")
	seedKeySupportSubscription(t, db, 302, "vip,svip")
	seedKeySupportSubscription(t, db, 302, "default")

	groups, err := ResolveKeyGroups(302)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip"}, groups)
}

// The base group is the fallback for an account without subscriptions, and it is
// still filtered through the same selectable-group check.
func TestResolveKeyGroupsFallsBackToBaseGroupWithoutSubscriptions(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 303, "default,vip,svip")

	groups, err := ResolveKeyGroups(303)
	require.NoError(t, err)
	assert.Equal(t, []string{"default"}, groups)
}

// The wallet allow-list grants reachability: a balance-only account whose own
// entitlement stops at default/vip must be able to route an "auto" key through
// the granted svip, even though svip is not selectable for it. The merged set is
// ordered by the configured priority, not by the order the list was written in.
func TestResolveKeyGroupsGrantsWalletGroupsToBalanceOnlyAccount(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	configureKeySupportWallet(t, []string{"default", "svip", "vip"}, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 401, "default")

	groups, err := ResolveKeyGroups(401)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip", "vip", "default"}, groups)
}

// EffectiveAutoGroups runs on every request, so the same grant must widen the
// request-time resolution, not only the stored snapshot.
func TestEffectiveAutoGroupsGrantsWalletGroups(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	configureKeySupportWallet(t, []string{"default", "svip", "vip"}, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 402, "default")

	assert.Equal(t, []string{"svip", "vip", "default"}, EffectiveAutoGroups(402, []string{"default"}))
}

// An empty allow-list is the "off" state and must leave the derivation exactly
// as it was before the grant existed: a balance-only account stays on its base
// group.
func TestResolveKeyGroupsUnchangedWhenWalletListEmpty(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	configureKeySupportWallet(t, nil, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 403, "default")

	groups, err := ResolveKeyGroups(403)
	require.NoError(t, err)
	assert.Equal(t, []string{"default"}, groups)
}

// A granted group still needs a group ratio, so a typo in the allow-list cannot
// point a key at a group that does not exist.
func TestResolveKeyGroupsSkipsWalletGroupWithoutRatio(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	configureKeySupportWallet(t, []string{"default", "ghost"}, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 404, "default")

	groups, err := ResolveKeyGroups(404)
	require.NoError(t, err)
	assert.Equal(t, []string{"default"}, groups)
}

// The per-user override is the administrator's escape hatch: an explicit list
// replaces the global grant, and an explicit empty list revokes it.
func TestResolveKeyGroupsHonoursPerUserWalletOverride(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	configureKeySupportWallet(t, []string{"default", "svip", "vip"}, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUserWithSetting(t, db, 405, "default", `{"wallet_balance_groups":"[]"}`)
	seedKeySupportUserWithSetting(t, db, 406, "default", `{"wallet_balance_groups":"[\"svip\"]"}`)

	revoked, err := ResolveKeyGroups(405)
	require.NoError(t, err)
	assert.Equal(t, []string{"default"}, revoked, "a per-user empty list must revoke the grant")

	narrowed, err := ResolveKeyGroups(406)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip"}, narrowed, "a per-user list must replace the global one")
}

// A stale wallet_balance_enabled key persisted by an older release is ignored:
// the allow-list is self-applying, so a non-empty list still grants even when
// the stored setting claims the feature is off.
func TestResolveKeyGroupsIgnoresStaleWalletEnabledKey(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	configureKeySupportWallet(t, []string{"svip"}, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUserWithSetting(t, db, 409, "default", `{"wallet_balance_enabled":false}`)

	groups, err := ResolveKeyGroups(409)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip"}, groups)
}

// Fusion only narrows the subscription part: the granted wallet groups are always
// kept, so a balance-only account is never left with the single subscribed group.
func TestResolveKeyGroupsKeepsWalletGrantsWhenFusionDisabled(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, false, 5)
	configureKeySupportWallet(t, []string{"svip"}, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 407, "default")
	seedKeySupportSubscription(t, db, 407, "vip")

	groups, err := ResolveKeyGroups(407)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip", "vip"}, groups)
}

// A group that is both subscribed and granted appears once, and the cap still
// applies after the two sources are merged and re-ordered.
func TestResolveKeyGroupsMergesSourcesWithoutDuplicatesAndCaps(t *testing.T) {
	configureProductionKeySupportGroups(t)
	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 5)
	configureKeySupportWallet(t, []string{"vip", "svip"}, nil)
	db := setupKeySupportTestDB(t)
	seedKeySupportUser(t, db, 408, "default")
	seedKeySupportSubscription(t, db, 408, "vip")

	groups, err := ResolveKeyGroups(408)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip", "vip"}, groups)

	configureKeySupportPolicy(t, []string{"svip", "vip", "default"}, true, 1)
	configureKeySupportWallet(t, []string{"default", "svip", "vip"}, nil)
	capped, err := ResolveKeyGroups(408)
	require.NoError(t, err)
	assert.Equal(t, []string{"svip"}, capped)
}
