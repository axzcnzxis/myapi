package model

import (
	"fmt"
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AdminDeleteSubscriptionPlan must only remove plans nothing references: an
// existing user subscription resolves its plan on every billing cycle (a missing
// plan would fail that user's requests), and a pending payment order still needs
// the plan to be completed. Referenced plans are refused, not cascaded.

func createSubscriptionOrder(t *testing.T, id, planId int, status string) {
	t.Helper()
	require.NoError(t, DB.Create(&SubscriptionOrder{
		Id:         id,
		UserId:     1,
		PlanId:     planId,
		Money:      1,
		TradeNo:    fmt.Sprintf("trade-%d", id),
		Status:     status,
		CreateTime: common.GetTimestamp(),
	}).Error)
}

func countPlans(t *testing.T, planId int) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", planId).Count(&count).Error)
	return count
}

func TestAdminDeleteSubscriptionPlan_DeletesUnreferencedPlan(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")
	for _, id := range []int{1, 2, 3} {
		createPlanWithId(t, id, 0)
	}

	require.NoError(t, AdminDeleteSubscriptionPlan(2))

	assert.Zero(t, countPlans(t, 2), "the plan row is gone")
	assert.Equal(t, map[int]int{1: 1, 3: 2}, planSeqById(listPlansWithSeq(t)),
		"the remaining plans tighten their numbers")
}

func TestAdminDeleteSubscriptionPlan_RefusesWhileUserSubscriptionsExist(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")
	resetSeqTable(t, &UserSubscription{}, "user_subscriptions")
	createPlanWithId(t, 1, 0)
	createSubscriptionWithId(t, 1, 1)
	createSubscriptionWithId(t, 1, 2)

	err := AdminDeleteSubscriptionPlan(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 条用户订阅")
	assert.Equal(t, int64(1), countPlans(t, 1), "the plan survives the refused deletion")

	var subs int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("plan_id = ?", 1).Count(&subs).Error)
	assert.Equal(t, int64(2), subs, "the referencing subscriptions are left untouched")
}

func TestAdminDeleteSubscriptionPlan_RefusesWhilePendingOrdersExist(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")
	resetSeqTable(t, &SubscriptionOrder{}, "subscription_orders")
	createPlanWithId(t, 1, 0)
	createSubscriptionOrder(t, 1, 1, common.TopUpStatusPending)

	err := AdminDeleteSubscriptionPlan(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未完成的支付订单")
	assert.Equal(t, int64(1), countPlans(t, 1), "the plan survives the refused deletion")
}

func TestAdminDeleteSubscriptionPlan_AllowsFinishedOrders(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")
	resetSeqTable(t, &SubscriptionOrder{}, "subscription_orders")
	createPlanWithId(t, 1, 0)
	createSubscriptionOrder(t, 1, 1, common.TopUpStatusSuccess)
	createSubscriptionOrder(t, 2, 1, common.TopUpStatusExpired)

	require.NoError(t, AdminDeleteSubscriptionPlan(1))

	assert.Zero(t, countPlans(t, 1), "completed and expired orders do not block deletion")
}

func TestAdminDeleteSubscriptionPlan_ReportsUnknownPlan(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")

	err := AdminDeleteSubscriptionPlan(999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "套餐不存在")
}

func TestAdminDeleteSubscriptionPlan_RejectsInvalidId(t *testing.T) {
	require.Error(t, AdminDeleteSubscriptionPlan(0))
}
