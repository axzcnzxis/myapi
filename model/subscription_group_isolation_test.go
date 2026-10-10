package model

import (
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedGroupIsolationUser(t *testing.T) *User {
	t.Helper()
	user := &User{
		Username: "group-isolation-user",
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "svip",
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func seedGroupIsolationPlan(t *testing.T, id int, group string, total int64) {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:            id,
		Title:         "plan-" + group,
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   total,
		UpgradeGroup:  group,
		Enabled:       true,
	}
	require.NoError(t, DB.Create(plan).Error)
}

func seedGroupIsolationSub(t *testing.T, id, userId, planId int, group string, total, used, endTime int64, allowOverflow bool) {
	t.Helper()
	sub := &UserSubscription{
		Id:                  id,
		UserId:              userId,
		PlanId:              planId,
		AmountTotal:         total,
		AmountUsed:          used,
		StartTime:           endTime - 30*24*3600,
		EndTime:             endTime,
		Status:              "active",
		UpgradeGroup:        group,
		AllowWalletOverflow: allowOverflow,
	}
	require.NoError(t, DB.Create(sub).Error)
}

// TestPreConsumeUserSubscriptionIsolatesByGroup 验证预扣费只落在请求分组对应的订阅上，
// 即使另一个分组存在更早到期的订阅（旧逻辑会选最早到期的那一个）。
func TestPreConsumeUserSubscriptionIsolatesByGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	user := seedGroupIsolationUser(t)

	seedGroupIsolationPlan(t, 9601, "", 1000)
	seedGroupIsolationPlan(t, 9602, "vip", 1000)
	seedGroupIsolationPlan(t, 9603, "svip", 1000)

	// default 订阅最早到期：旧逻辑会优先命中它。
	seedGroupIsolationSub(t, 9601, user.Id, 9601, "", 1000, 0, now+100, true)
	seedGroupIsolationSub(t, 9602, user.Id, 9602, "vip", 1000, 0, now+200, true)
	seedGroupIsolationSub(t, 9603, user.Id, 9603, "svip", 1000, 0, now+300, true)

	cases := []struct {
		requestId string
		group     string
		wantSub   int
	}{
		{"req-vip", "vip", 9602},
		{"req-svip", "svip", 9603},
		{"req-default", "default", 9601},
		{"req-blank", "", 9601}, // 空分组视为 default
	}
	for _, tc := range cases {
		res, err := PreConsumeUserSubscription(tc.requestId, user.Id, "test-model", tc.group, 0, 100)
		require.NoError(t, err, "group=%q", tc.group)
		assert.Equal(t, tc.wantSub, res.UserSubscriptionId, "group=%q", tc.group)
		assert.Equal(t, int64(100), res.PreConsumed)
	}

	// default 订阅被 req-default 与 req-blank 各扣一次；vip/svip 各扣一次。
	wantUsed := map[int]int64{9601: 200, 9602: 100, 9603: 100}
	for id, want := range wantUsed {
		assert.Equal(t, want, getSubscriptionResetSub(t, id).AmountUsed, "sub %d", id)
	}
}

// TestPreConsumeUserSubscriptionRejectsOtherGroupQuota 验证请求分组额度不足时不会借用
// 其它分组的订阅额度，而是直接报额度不足。
func TestPreConsumeUserSubscriptionRejectsOtherGroupQuota(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	user := seedGroupIsolationUser(t)

	seedGroupIsolationPlan(t, 9611, "", 100000)
	seedGroupIsolationPlan(t, 9612, "vip", 50)
	seedGroupIsolationSub(t, 9611, user.Id, 9611, "", 100000, 0, now+100, true)
	seedGroupIsolationSub(t, 9612, user.Id, 9612, "vip", 50, 0, now+200, true)

	_, err := PreConsumeUserSubscription("req-vip-insufficient", user.Id, "test-model", "vip", 0, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscription quota insufficient")

	// default 订阅未被触碰。
	assert.Equal(t, int64(0), getSubscriptionResetSub(t, 9611).AmountUsed)
}

// TestPreConsumeUserSubscriptionNoMatchingGroup 验证请求分组没有对应订阅时报未配置订阅。
func TestPreConsumeUserSubscriptionNoMatchingGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	user := seedGroupIsolationUser(t)

	seedGroupIsolationPlan(t, 9621, "", 100000)
	seedGroupIsolationSub(t, 9621, user.Id, 9621, "", 100000, 0, now+100, true)

	_, err := PreConsumeUserSubscription("req-svip-none", user.Id, "test-model", "svip", 0, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active subscription")
	assert.Equal(t, int64(0), getSubscriptionResetSub(t, 9621).AmountUsed)
}

// TestHasActiveUserSubscriptionScopedByGroup 验证活跃订阅检查按分组收敛。
func TestHasActiveUserSubscriptionScopedByGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	user := seedGroupIsolationUser(t)

	seedGroupIsolationPlan(t, 9631, "vip", 1000)
	seedGroupIsolationSub(t, 9631, user.Id, 9631, "vip", 1000, 0, now+100, true)

	hasVip, err := HasActiveUserSubscription(user.Id, "vip")
	require.NoError(t, err)
	assert.True(t, hasVip)

	for _, group := range []string{"svip", "default", ""} {
		has, err := HasActiveUserSubscription(user.Id, group)
		require.NoError(t, err)
		assert.False(t, has, "group=%q", group)
	}
}

// TestUserActiveSubscriptionsAllowWalletOverflowScopedByGroup 验证钱包回退开关按分组判定。
func TestUserActiveSubscriptionsAllowWalletOverflowScopedByGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	user := seedGroupIsolationUser(t)

	seedGroupIsolationPlan(t, 9641, "", 1000)
	seedGroupIsolationPlan(t, 9642, "vip", 1000)
	seedGroupIsolationSub(t, 9641, user.Id, 9641, "", 1000, 0, now+100, true)
	seedGroupIsolationSub(t, 9642, user.Id, 9642, "vip", 1000, 0, now+200, false)

	allowDefault, err := UserActiveSubscriptionsAllowWalletOverflow(user.Id, "default")
	require.NoError(t, err)
	assert.True(t, allowDefault, "default 分组的订阅允许回退钱包")

	allowVip, err := UserActiveSubscriptionsAllowWalletOverflow(user.Id, "vip")
	require.NoError(t, err)
	assert.False(t, allowVip, "vip 分组的订阅禁止回退钱包")

	// 没有任何订阅的分组默认允许（不阻断钱包路径）。
	allowSvip, err := UserActiveSubscriptionsAllowWalletOverflow(user.Id, "svip")
	require.NoError(t, err)
	assert.True(t, allowSvip)
}

// TestPreConsumeUserSubscriptionReportsExhaustedQuota 验证请求分组的订阅余额为 0 时返回
// 专用提示（余额为 0，请联系管理员重置额度套餐），而不是笼统的额度不足。
func TestPreConsumeUserSubscriptionReportsExhaustedQuota(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	user := seedGroupIsolationUser(t)

	seedGroupIsolationPlan(t, 9665, "vip", 100)
	seedGroupIsolationSub(t, 9665, user.Id, 9665, "vip", 100, 100, now+100, true)

	_, err := PreConsumeUserSubscription("req-vip-exhausted", user.Id, "test-model", "vip", 0, 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubscriptionQuotaExhausted)
	assert.Equal(t, "余额为 0，请联系管理员重置额度套餐", err.Error())

	// 同分组还有另一个仍有余额、但不够本次请求的订阅时，按普通额度不足提示。
	seedGroupIsolationPlan(t, 9666, "vip", 1000)
	seedGroupIsolationSub(t, 9666, user.Id, 9666, "vip", 1000, 950, now+200, true)

	_, err = PreConsumeUserSubscription("req-vip-partial", user.Id, "test-model", "vip", 0, 100)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrSubscriptionQuotaExhausted)
	assert.Contains(t, err.Error(), "subscription quota insufficient")
}
