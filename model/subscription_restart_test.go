package model

import (
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminRestartUserSubscriptionExtendsDurationAddsAmountAndRestoresGroup(t *testing.T) {
	truncateTables(t)
	useUserCacheMiniRedis(t)
	now := GetDBTimestamp()

	user := User{
		Username: "restart-user",
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, DB.Create(&user).Error)

	plan := &SubscriptionPlan{
		Id:            9701,
		Title:         "Restart Pro",
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   1000,
		UpgradeGroup:  "pro",
		Enabled:       true,
	}
	require.NoError(t, DB.Create(plan).Error)

	// A cancelled subscription whose end_time was zeroed to "now" by invalidate,
	// and whose user group had been downgraded back to default.
	sub := &UserSubscription{
		Id:            9701,
		UserId:        user.Id,
		PlanId:        plan.Id,
		AmountTotal:   1000,
		AmountUsed:    400,
		StartTime:     now - 30*24*3600,
		EndTime:       now,
		Status:        "cancelled",
		UpgradeGroup:  "pro",
		PrevUserGroup: "default",
	}
	require.NoError(t, DB.Create(sub).Error)

	msg, err := AdminRestartUserSubscription(sub.Id, true, 500)
	require.NoError(t, err)
	assert.Contains(t, msg, "pro")

	got := getSubscriptionResetSub(t, sub.Id)
	assert.Equal(t, "active", got.Status)
	assert.Greater(t, got.EndTime, now+27*24*3600)
	assert.Less(t, got.EndTime, now+32*24*3600)
	assert.Equal(t, int64(1500), got.AmountTotal)
	assert.Equal(t, int64(400), got.AmountUsed)

	var updatedUser User
	require.NoError(t, DB.First(&updatedUser, user.Id).Error)
	assert.Equal(t, "pro", updatedUser.Group)
}

func TestAdminRestartUserSubscriptionWithoutExtendKeepsEndTimeAndRejectsNegativeAmount(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	plan := &SubscriptionPlan{
		Id:            9702,
		Title:         "Restart Basic",
		PriceAmount:   1,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   100,
		Enabled:       true,
	}
	require.NoError(t, DB.Create(plan).Error)

	sub := &UserSubscription{
		Id:          9702,
		UserId:      501,
		PlanId:      plan.Id,
		AmountTotal: 100,
		StartTime:   now - 3600,
		EndTime:     now - 10,
		Status:      "cancelled",
	}
	require.NoError(t, DB.Create(sub).Error)

	_, err := AdminRestartUserSubscription(sub.Id, false, -1)
	require.Error(t, err)

	_, err = AdminRestartUserSubscription(sub.Id, false, 0)
	require.NoError(t, err)

	got := getSubscriptionResetSub(t, sub.Id)
	assert.Equal(t, "active", got.Status)
	assert.Equal(t, sub.EndTime, got.EndTime)
	assert.Equal(t, int64(100), got.AmountTotal)
}

func TestAdminRestartUserSubscriptionRejectsInvalidId(t *testing.T) {
	truncateTables(t)
	_, err := AdminRestartUserSubscription(0, true, 0)
	require.Error(t, err)
}
