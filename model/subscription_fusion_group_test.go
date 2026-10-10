package model

import (
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedFusionPlan(t *testing.T, id int, upgradeGroup string, total int64) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:            id,
		Title:         "fusion-plan-" + upgradeGroup,
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   total,
		UpgradeGroup:  upgradeGroup,
		Enabled:       true,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

// TestNormalizeSubscriptionGroups 验证分组串的规范化：去空白、去重、保序，空值归零。
func TestNormalizeSubscriptionGroups(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{",", ""},
		{"vip", "vip"},
		{"  vip  ", "vip"},
		{"default,vip", "default,vip"},
		{"vip,default", "vip,default"},
		{" default , vip ", "default,vip"},
		{"vip,,svip", "vip,svip"},
		{"vip,vip,default", "vip,default"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, NormalizeSubscriptionGroups(tc.raw), "raw=%q", tc.raw)
	}
}

// TestSubscriptionGroupMatchesFusion 验证融合订阅匹配其服务的每个分组，且不匹配未列出的分组。
func TestSubscriptionGroupMatchesFusion(t *testing.T) {
	fusion := &UserSubscription{UpgradeGroup: "vip,default"}
	assert.True(t, subscriptionGroupMatches(fusion, "vip"))
	assert.True(t, subscriptionGroupMatches(fusion, "default"))
	assert.True(t, subscriptionGroupMatches(fusion, ""), "空请求分组视为 default")
	assert.False(t, subscriptionGroupMatches(fusion, "svip"))

	legacy := &UserSubscription{UpgradeGroup: ""}
	assert.True(t, subscriptionGroupMatches(legacy, "default"))
	assert.True(t, subscriptionGroupMatches(legacy, ""))
	assert.False(t, subscriptionGroupMatches(legacy, "vip"))

	triple := &UserSubscription{UpgradeGroup: "default,vip,svip"}
	for _, g := range []string{"default", "vip", "svip", ""} {
		assert.True(t, subscriptionGroupMatches(triple, g), "group=%q", g)
	}
}

// TestPreConsumeFusionSubscriptionServesAllItsGroups 验证一个融合订阅同时服务 default 与 vip：
// 两个分组的请求都扣在同一订阅上，未列出的分组不借用。
func TestPreConsumeFusionSubscriptionServesAllItsGroups(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	user := seedGroupIsolationUser(t)

	seedFusionPlan(t, 9651, "vip,default", 1000)
	seedGroupIsolationSub(t, 9651, user.Id, 9651, "vip,default", 1000, 0, now+100, true)

	for _, tc := range []struct {
		requestId string
		group     string
	}{
		{"req-fusion-vip", "vip"},
		{"req-fusion-default", "default"},
		{"req-fusion-blank", ""}, // 空分组视为 default
	} {
		res, err := PreConsumeUserSubscription(tc.requestId, user.Id, "test-model", tc.group, 0, 100)
		require.NoError(t, err, "group=%q", tc.group)
		assert.Equal(t, 9651, res.UserSubscriptionId, "group=%q", tc.group)
	}
	assert.Equal(t, int64(300), getSubscriptionResetSub(t, 9651).AmountUsed)

	// 未列出的 svip 分组没有对应订阅，不借用该融合订阅的额度。
	_, err := PreConsumeUserSubscription("req-fusion-svip", user.Id, "test-model", "svip", 0, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active subscription")
	assert.Equal(t, int64(300), getSubscriptionResetSub(t, 9651).AmountUsed)
}

// TestCreateUserSubscriptionFusionPlanKeepsUserGroup 验证创建融合订阅本身只记录购买前的
// 分组快照，用户分组在同步后才跟随套餐的多个分组。
func TestCreateUserSubscriptionFusionPlanKeepsUserGroup(t *testing.T) {
	truncateTables(t)
	user := seedGroupIsolationUser(t) // group = svip
	plan := seedFusionPlan(t, 9661, "default,vip", 1000)

	// The shared test harness (TestMain) pins the sqlite pool to a single
	// connection, and CreateUserSubscriptionFromPlanTx reads the DB clock via the
	// global handle; holding an explicit transaction here would occupy that one
	// connection and block the clock read. Call it on the handle directly.
	sub, err := CreateUserSubscriptionFromPlanTx(DB, user.Id, plan, "test")
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, "default,vip", sub.UpgradeGroup)
	assert.Equal(t, "svip", sub.PrevUserGroup, "购买前的分组快照仅供参考")

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, "svip", got.Group, "创建订阅本身不应改动用户分组")

	// 同步后用户分组跟随套餐分组：套餐 N 个分组 → 用户 N 个分组。
	changed, err := SyncUserGroupWithSubscriptionsTx(DB, user.Id)
	require.NoError(t, err)
	assert.Equal(t, "default,vip", changed)
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, "default,vip", got.Group)

	// 订阅同时服务 default 与 vip。
	assert.True(t, subscriptionGroupMatches(sub, "default"))
	assert.True(t, subscriptionGroupMatches(sub, "vip"))
	assert.False(t, subscriptionGroupMatches(sub, "svip"))
}

// TestCreateUserSubscriptionSingleGroupPlanUpgradesUser 验证单分组套餐在同步后把用户分组
// 升级为套餐分组。
func TestCreateUserSubscriptionSingleGroupPlanUpgradesUser(t *testing.T) {
	truncateTables(t)
	user := seedGroupIsolationUser(t) // group = svip
	plan := seedFusionPlan(t, 9671, "vip", 1000)

	// The shared test harness (TestMain) pins the sqlite pool to a single
	// connection, and CreateUserSubscriptionFromPlanTx reads the DB clock via the
	// global handle; holding an explicit transaction here would occupy that one
	// connection and block the clock read. Call it on the handle directly.
	sub, err := CreateUserSubscriptionFromPlanTx(DB, user.Id, plan, "test")
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, "vip", sub.UpgradeGroup)
	assert.Equal(t, "svip", sub.PrevUserGroup, "应记录升级前的分组快照")

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, "svip", got.Group, "创建订阅本身不应改动用户分组")

	changed, err := SyncUserGroupWithSubscriptionsTx(DB, user.Id)
	require.NoError(t, err)
	assert.Equal(t, "vip", changed)
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, "vip", got.Group, "同步后单分组套餐应把用户升级到套餐分组")
}

// TestDowngradeFallsBackToRemainingSubscriptionGroups 验证订阅被作废/删除后按
// 「剩余有效订阅服务分组的并集」回退，没有剩余有效订阅时回退到 default。
func TestDowngradeFallsBackToRemainingSubscriptionGroups(t *testing.T) {
	cases := []struct {
		name        string
		invalidated string   // 被作废订阅的 upgrade_group
		remaining   []string // 仍然有效的订阅分组，越靠前剩余有效期越长
		want        string
	}{
		{"融合+vip：作废融合后回退到 vip", "default,vip", []string{"vip"}, "vip"},
		{"svip+vip：作废 svip 后回退到 vip", "svip", []string{"vip"}, "vip"},
		{"多个剩余订阅取并集并按剩余有效期排序", "vip", []string{"svip", "default"}, "svip,default"},
		{"没有剩余有效订阅时回退 default", "default,vip", nil, "default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			now := GetDBTimestamp()
			user := seedGroupIsolationUser(t) // group = svip

			// 被作废的订阅：与 AdminInvalidateUserSubscription 一致，status=cancelled、end_time=now。
			seedFusionPlan(t, 9681, tc.invalidated, 1000)
			require.NoError(t, DB.Create(&UserSubscription{
				Id:           9681,
				UserId:       user.Id,
				PlanId:       9681,
				AmountTotal:  1000,
				StartTime:    now - 30*24*3600,
				EndTime:      now,
				Status:       "cancelled",
				UpgradeGroup: tc.invalidated,
			}).Error)

			for i, group := range tc.remaining {
				id := 9690 + i
				seedFusionPlan(t, id, group, 1000)
				seedGroupIsolationSub(t, id, user.Id, id, group, 1000, 0, now+int64(300-i*100), true)
			}

			invalidated := getSubscriptionResetSub(t, 9681)
			require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
				target, err := downgradeUserGroupForSubscriptionTx(tx, &invalidated, now)
				if err != nil {
					return err
				}
				assert.Equal(t, tc.want, target)
				return nil
			}))

			var got User
			require.NoError(t, DB.First(&got, user.Id).Error)
			assert.Equal(t, tc.want, got.Group)
		})
	}
}

// TestExpireDueSubscriptionsFallsBackToRemainingSubscriptionGroups 验证套餐到期后自动回退到
// 剩余有效订阅服务分组的并集，全部到期则回退 default。
func TestExpireDueSubscriptionsFallsBackToRemainingSubscriptionGroups(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	// 融合套餐到期，另一个 vip 订阅仍然有效。
	user := seedGroupIsolationUser(t)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("group", "default,vip").Error)

	seedFusionPlan(t, 9701, "default,vip", 1000)
	seedFusionPlan(t, 9702, "vip", 1000)
	seedGroupIsolationSub(t, 9701, user.Id, 9701, "default,vip", 1000, 0, now-10, true)
	seedGroupIsolationSub(t, 9702, user.Id, 9702, "vip", 1000, 0, now+1000, true)

	expired, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expired)

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, "vip", got.Group, "应回退到剩余有效订阅服务的分组")

	// 最后一个有效订阅也到期后回退 default。
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", 9702).
		Updates(map[string]any{"end_time": now - 1}).Error)
	expired, err = ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expired)
	require.NoError(t, DB.First(&got, user.Id).Error)
	assert.Equal(t, "default", got.Group, "没有剩余有效订阅时回退 default")
}

// TestDowngradeSingleGroupSubscriptionRevertsUserGroup 验证单分组订阅过期时仍按升级前分组回退。
func TestDowngradeSingleGroupSubscriptionRevertsUserGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	user := &User{
		Username: "downgrade-single-user",
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "vip",
	}
	require.NoError(t, DB.Create(user).Error)

	seedFusionPlan(t, 9691, "vip", 1000)
	sub := &UserSubscription{
		Id:            9691,
		UserId:        user.Id,
		PlanId:        9691,
		AmountTotal:   1000,
		StartTime:     now - 30*24*3600,
		EndTime:       now - 10,
		Status:        "expired",
		UpgradeGroup:  "vip",
		PrevUserGroup: "default",
	}
	require.NoError(t, DB.Create(sub).Error)

	got := getSubscriptionResetSub(t, 9691)
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		target, err := downgradeUserGroupForSubscriptionTx(tx, &got, now)
		assert.Equal(t, "default", target)
		return err
	}))

	var updated User
	require.NoError(t, DB.First(&updated, user.Id).Error)
	assert.Equal(t, "default", updated.Group)
}
