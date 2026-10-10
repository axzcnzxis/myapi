package model

import (
	"fmt"
	"testing"

	"github.com/axzcnzxis/myapi/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The console numbers every admin list by creation order: the first column shows
// the row's 1-based position among the rows the listing can return, so deleting
// a row tightens the numbers below it and a new row always takes the next one.
// These tests cover the derived sequence behind the users, redemption codes,
// models and vendors lists (see seqByID and the Attach*Seq helpers).

// resetSeqTable empties one table and rewinds SQLite's AUTOINCREMENT counter, so
// each test starts from a known state and "a new row appends at the end" holds.
// Deleted rows are removed for good, which is what a fresh table means here.
func resetSeqTable(t *testing.T, model any, table string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(model))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(model).Error)
	DB.Exec("DELETE FROM sqlite_sequence WHERE name = ?", table)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(model).Error)
	})
}

func createUserWithId(t *testing.T, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       id,
		Username: fmt.Sprintf("user-%d", id),
		Password: "password123",
		AffCode:  fmt.Sprintf("aff-%d", id),
	}).Error)
}

func createRedemptionWithId(t *testing.T, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&Redemption{
		Id:          id,
		Key:         fmt.Sprintf("%032d", id),
		Name:        fmt.Sprintf("code-%d", id),
		Status:      common.RedemptionCodeStatusEnabled,
		CreatedTime: common.GetTimestamp(),
	}).Error)
}

func createModelWithId(t *testing.T, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&Model{
		Id:          id,
		ModelName:   fmt.Sprintf("model-%d", id),
		CreatedTime: common.GetTimestamp(),
	}).Error)
}

func createVendorWithId(t *testing.T, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&Vendor{
		Id:          id,
		Name:        fmt.Sprintf("vendor-%d", id),
		CreatedTime: common.GetTimestamp(),
	}).Error)
}

func listUsersWithSeq(t *testing.T) []*User {
	t.Helper()
	users, _, err := GetAllUsers(&common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.NoError(t, AttachUserSeq(DB, users))
	return users
}

func listRedemptionsWithSeq(t *testing.T) []*Redemption {
	t.Helper()
	redemptions, _, err := GetAllRedemptions(0, 10)
	require.NoError(t, err)
	require.NoError(t, AttachRedemptionSeq(DB, redemptions))
	return redemptions
}

func userSeqById(users []*User) map[int]int {
	seqs := make(map[int]int, len(users))
	for _, user := range users {
		seqs[user.Id] = user.Seq
	}
	return seqs
}

func redemptionSeqById(redemptions []*Redemption) map[int]int {
	seqs := make(map[int]int, len(redemptions))
	for _, redemption := range redemptions {
		seqs[redemption.Id] = redemption.Seq
	}
	return seqs
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

func TestAttachUserSeq_NumbersUsersByAscendingId(t *testing.T) {
	resetSeqTable(t, &User{}, "users")
	for _, id := range []int{1, 2, 3} {
		createUserWithId(t, id)
	}

	assert.Equal(t, map[int]int{1: 1, 2: 2, 3: 3}, userSeqById(listUsersWithSeq(t)))
}

func TestGetAllUsers_DefaultListingIsOldestFirst(t *testing.T) {
	resetSeqTable(t, &User{}, "users")
	for _, id := range []int{1, 2, 3} {
		createUserWithId(t, id)
	}

	// Without any explicit sort the console must list users oldest-first, so the
	// sequence column reads 1, 2, 3 from top to bottom.
	users := listUsersWithSeq(t)
	require.Len(t, users, 3)
	for i, user := range users {
		assert.Equal(t, i+1, user.Id, "rows must come back oldest first")
		assert.Equal(t, i+1, user.Seq)
	}
}

func TestAttachUserSeq_KeepsTheNumberOfADeletedUser(t *testing.T) {
	resetSeqTable(t, &User{}, "users")
	for _, id := range []int{1, 2, 3} {
		createUserWithId(t, id)
	}

	// The user listing is unscoped: a deleted user stays visible (flagged as
	// deleted), so it keeps its number instead of freeing it for someone else.
	require.NoError(t, DB.Delete(&User{}, 2).Error)

	users, total, err := GetAllUsers(&common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total, "deleted users are still listed")
	require.NoError(t, AttachUserSeq(DB, users))
	assert.Equal(t, map[int]int{1: 1, 2: 2, 3: 3}, userSeqById(users))
}

func TestAttachUserSeq_ExplicitSortStillWins(t *testing.T) {
	resetSeqTable(t, &User{}, "users")
	for _, id := range []int{1, 2, 3} {
		createUserWithId(t, id)
	}

	users, _, err := GetAllUsers(&common.PageInfo{Page: 1, PageSize: 10}, NewUserSortOptions("id", "desc"))
	require.NoError(t, err)
	require.NoError(t, AttachUserSeq(DB, users))

	// Sorting newest-first must not renumber anyone: the sequence always reports
	// the creation order.
	require.Len(t, users, 3)
	for i, user := range users {
		assert.Equal(t, 3-i, user.Id)
		assert.Equal(t, 3-i, user.Seq)
	}
}

func TestAttachUserSeq_NumbersAPageByGlobalPosition(t *testing.T) {
	resetSeqTable(t, &User{}, "users")
	for _, id := range []int{1, 2, 3, 4, 5} {
		createUserWithId(t, id)
	}

	users, _, err := GetAllUsers(&common.PageInfo{Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.NoError(t, AttachUserSeq(DB, users))
	require.Len(t, users, 2)
	assert.Equal(t, map[int]int{3: 3, 4: 4}, userSeqById(users))
}

// ---------------------------------------------------------------------------
// Redemption codes
// ---------------------------------------------------------------------------

func TestGetAllRedemptions_DefaultListingIsOldestFirst(t *testing.T) {
	resetSeqTable(t, &Redemption{}, "redemptions")
	for _, id := range []int{1, 2, 3} {
		createRedemptionWithId(t, id)
	}

	redemptions := listRedemptionsWithSeq(t)
	require.Len(t, redemptions, 3)
	for i, redemption := range redemptions {
		assert.Equal(t, i+1, redemption.Id, "rows must come back oldest first")
		assert.Equal(t, i+1, redemption.Seq)
	}
}

func TestAttachRedemptionSeq_DeletingACodeRenumbersTheRest(t *testing.T) {
	resetSeqTable(t, &Redemption{}, "redemptions")
	for _, id := range []int{1, 2, 3} {
		createRedemptionWithId(t, id)
	}

	// The listing hides deleted codes, so the deleted one frees its number and
	// the former #3 moves up to #2.
	require.NoError(t, DB.Delete(&Redemption{}, 2).Error)

	redemptions, total, err := GetAllRedemptions(0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.NoError(t, AttachRedemptionSeq(DB, redemptions))
	assert.Equal(t, map[int]int{1: 1, 3: 2}, redemptionSeqById(redemptions))
}

func TestAttachRedemptionSeq_NewCodeAppendsAtTheEnd(t *testing.T) {
	resetSeqTable(t, &Redemption{}, "redemptions")
	for _, id := range []int{1, 2, 3} {
		createRedemptionWithId(t, id)
	}
	require.NoError(t, DB.Delete(&Redemption{}, 2).Error)

	createRedemptionWithId(t, 4)
	assert.Equal(t, map[int]int{1: 1, 3: 2, 4: 3}, redemptionSeqById(listRedemptionsWithSeq(t)))
}

// ---------------------------------------------------------------------------
// Models and vendors
// ---------------------------------------------------------------------------

func TestAttachModelSeq_NumbersRecordsAndSkipsDerivedRows(t *testing.T) {
	resetSeqTable(t, &Model{}, "models")
	createModelWithId(t, 1)
	createModelWithId(t, 2)

	models := []*Model{{Id: 1}, {Id: 2}, {ModelName: "served-by-a-channel"}}
	require.NoError(t, AttachModelSeq(DB, models))

	assert.Equal(t, 1, models[0].Seq)
	assert.Equal(t, 2, models[1].Seq)
	assert.Equal(t, 0, models[2].Seq, "a row without a stored record has no number")
}

func TestAttachModelSeq_IgnoresDeletedRecords(t *testing.T) {
	resetSeqTable(t, &Model{}, "models")
	for _, id := range []int{1, 2, 3} {
		createModelWithId(t, id)
	}
	require.NoError(t, DB.Delete(&Model{}, 2).Error)

	models, total, err := SearchModels("", "", "", "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.NoError(t, AttachModelSeq(DB, models))

	seqs := make(map[int]int, len(models))
	for _, metadata := range models {
		seqs[metadata.Id] = metadata.Seq
	}
	assert.Equal(t, map[int]int{1: 1, 3: 2}, seqs)
}

func TestSearchVendors_DefaultListingIsOldestFirstWithSeq(t *testing.T) {
	resetSeqTable(t, &Vendor{}, "vendors")
	resetSeqTable(t, &Model{}, "models")
	for _, id := range []int{1, 2, 3} {
		createVendorWithId(t, id)
	}

	vendors, total, err := SearchVendors("", 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.NoError(t, AttachVendorSeq(DB, vendors))

	require.Len(t, vendors, 3)
	for i, vendor := range vendors {
		assert.Equal(t, i+1, vendor.Id, "rows must come back oldest first")
		assert.Equal(t, i+1, vendor.Seq)
	}
}

// ---------------------------------------------------------------------------
// User subscriptions
// ---------------------------------------------------------------------------

func createSubscriptionWithId(t *testing.T, userId, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&UserSubscription{
		Id:        id,
		UserId:    userId,
		PlanId:    1,
		Status:    "active",
		StartTime: common.GetTimestamp(),
		EndTime:   common.GetTimestamp() + 86400,
	}).Error)
}

func subscriptionSeqById(subs []SubscriptionSummary) map[int]int {
	seqs := make(map[int]int, len(subs))
	for _, s := range subs {
		seqs[s.Subscription.Id] = s.Subscription.Seq
	}
	return seqs
}

func listUserSubscriptionsWithSeq(t *testing.T, userId int) []SubscriptionSummary {
	t.Helper()
	subs, err := GetAllUserSubscriptionsByCreation(userId)
	require.NoError(t, err)
	return subs
}

func TestGetAllUserSubscriptionsByCreation_NumbersEachUserIndependently(t *testing.T) {
	resetSeqTable(t, &UserSubscription{}, "user_subscriptions")
	for _, id := range []int{1, 2, 3} {
		createSubscriptionWithId(t, 1, id)
	}
	for _, id := range []int{4, 5} {
		createSubscriptionWithId(t, 2, id)
	}

	// The listing must come back oldest-first so the ID column reads 1, 2, 3
	// from top to bottom.
	user1 := listUserSubscriptionsWithSeq(t, 1)
	require.Len(t, user1, 3)
	for i, s := range user1 {
		assert.Equal(t, i+1, s.Subscription.Id, "rows must come back oldest first")
		assert.Equal(t, i+1, s.Subscription.Seq)
	}

	// Numbering is per user: the second user restarts at 1 and does not inherit
	// the first user's rows.
	assert.Equal(t, map[int]int{4: 1, 5: 2}, subscriptionSeqById(listUserSubscriptionsWithSeq(t, 2)))
}

func TestGetAllUserSubscriptionsByCreation_DeletingTightensAndNewAppends(t *testing.T) {
	resetSeqTable(t, &UserSubscription{}, "user_subscriptions")
	for _, id := range []int{1, 2, 3, 4} {
		createSubscriptionWithId(t, 1, id)
	}

	// Subscriptions are hard-deleted, so the removed #2 frees its number and the
	// rows below it move up.
	require.NoError(t, DB.Where("id = ?", 2).Delete(&UserSubscription{}).Error)
	assert.Equal(t, map[int]int{1: 1, 3: 2, 4: 3}, subscriptionSeqById(listUserSubscriptionsWithSeq(t, 1)))

	// A new subscription always takes the next number at the end.
	createSubscriptionWithId(t, 1, 5)
	assert.Equal(t, map[int]int{1: 1, 3: 2, 4: 3, 5: 4}, subscriptionSeqById(listUserSubscriptionsWithSeq(t, 1)))
}

func TestGetAllUserSubscriptions_KeepsItsOwnOrderAndNumbersByCreation(t *testing.T) {
	resetSeqTable(t, &UserSubscription{}, "user_subscriptions")
	now := common.GetTimestamp()
	// id 3 ends latest, id 1 ends soonest: the self view lists the longest-running
	// subscription first, while the number shown follows creation order, so it
	// matches the admin dialog's ID column.
	require.NoError(t, DB.Create(&UserSubscription{Id: 1, UserId: 1, PlanId: 1, Status: "active", StartTime: now, EndTime: now + 1000}).Error)
	require.NoError(t, DB.Create(&UserSubscription{Id: 2, UserId: 1, PlanId: 1, Status: "active", StartTime: now, EndTime: now + 2000}).Error)
	require.NoError(t, DB.Create(&UserSubscription{Id: 3, UserId: 1, PlanId: 1, Status: "active", StartTime: now, EndTime: now + 3000}).Error)

	subs, err := GetAllUserSubscriptions(1)
	require.NoError(t, err)
	require.Len(t, subs, 3)
	ids := make([]int, 0, len(subs))
	for _, s := range subs {
		ids = append(ids, s.Subscription.Id)
	}
	assert.Equal(t, []int{3, 2, 1}, ids, "self view stays longest-running first")
	assert.Equal(t, map[int]int{1: 1, 2: 2, 3: 3}, subscriptionSeqById(subs),
		"the self view numbers rows by creation order, like the admin dialog")
}

func TestGetAllActiveUserSubscriptions_NumbersEachUserIndependently(t *testing.T) {
	resetSeqTable(t, &UserSubscription{}, "user_subscriptions")
	for _, id := range []int{1, 2, 3} {
		createSubscriptionWithId(t, 1, id)
	}
	for _, id := range []int{4, 5} {
		createSubscriptionWithId(t, 2, id)
	}

	active, err := GetAllActiveUserSubscriptions(2)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, map[int]int{4: 1, 5: 2}, subscriptionSeqById(active))
}

// ---------------------------------------------------------------------------
// Subscription plans
// ---------------------------------------------------------------------------

func createPlanWithId(t *testing.T, id int, sortOrder int) {
	t.Helper()
	require.NoError(t, DB.Create(&SubscriptionPlan{
		Id:        id,
		Title:     fmt.Sprintf("plan-%d", id),
		SortOrder: sortOrder,
	}).Error)
}

func planSeqById(plans []SubscriptionPlan) map[int]int {
	seqs := make(map[int]int, len(plans))
	for _, plan := range plans {
		seqs[plan.Id] = plan.Seq
	}
	return seqs
}

// listPlansWithSeq runs the admin listing's query.
func listPlansWithSeq(t *testing.T) []SubscriptionPlan {
	t.Helper()
	var plans []SubscriptionPlan
	require.NoError(t, DB.Order("id asc").Find(&plans).Error)
	require.NoError(t, AttachSubscriptionPlanSeq(DB, plans))
	return plans
}

func TestAttachSubscriptionPlanSeq_NumbersPlansByCreationNotPriority(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")
	// Plan 3 has the highest priority (sort_order), which only affects the order
	// customers see; the console still numbers plans by creation order.
	createPlanWithId(t, 1, 0)
	createPlanWithId(t, 2, 0)
	createPlanWithId(t, 3, 10)

	plans := listPlansWithSeq(t)
	require.Len(t, plans, 3)
	for i, plan := range plans {
		assert.Equal(t, i+1, plan.Id, "rows must come back oldest first")
		assert.Equal(t, i+1, plan.Seq)
	}
}

func TestAttachSubscriptionPlanSeq_DeletingAPlanTightensTheRest(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")
	for _, id := range []int{1, 2, 3, 4, 5} {
		createPlanWithId(t, id, 0)
	}

	// Plans are hard-deleted, so removing #3 frees its number: the former #4
	// becomes #3 and the former #5 becomes #4.
	require.NoError(t, DB.Delete(&SubscriptionPlan{}, 3).Error)
	assert.Equal(t, map[int]int{1: 1, 2: 2, 4: 3, 5: 4}, planSeqById(listPlansWithSeq(t)))
}

func TestAttachSubscriptionPlanSeq_NewPlanAppendsAtTheEnd(t *testing.T) {
	resetSeqTable(t, &SubscriptionPlan{}, "subscription_plans")
	for _, id := range []int{1, 2, 3, 4, 5} {
		createPlanWithId(t, id, 0)
	}
	require.NoError(t, DB.Delete(&SubscriptionPlan{}, 3).Error)

	createPlanWithId(t, 6, 0)
	assert.Equal(t, map[int]int{1: 1, 2: 2, 4: 3, 5: 4, 6: 5},
		planSeqById(listPlansWithSeq(t)), "a new plan always takes the next number")
}

// ---------------------------------------------------------------------------
// Empty pages
// ---------------------------------------------------------------------------

func TestAttachSeq_LeavesEmptyListingsAlone(t *testing.T) {
	require.NoError(t, AttachUserSeq(DB, nil))
	require.NoError(t, AttachRedemptionSeq(DB, nil))
	require.NoError(t, AttachModelSeq(DB, nil))
	require.NoError(t, AttachVendorSeq(DB, nil))
	require.NoError(t, AttachUserSubscriptionSeq(DB, nil))
	require.NoError(t, AttachSubscriptionPlanSeq(DB, nil))
}
