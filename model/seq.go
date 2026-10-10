package model

import (
	"fmt"

	"gorm.io/gorm"
)

// The console shows a "sequence" number instead of the raw primary key on every
// admin list (channels, users, redemption codes, models, vendors, subscription
// plans and user subscriptions). A sequence is the row's 1-based position among
// the rows the listing can return, ordered by ascending id: it therefore reads
// 1, 2, 3 from top to bottom, deleting a row moves every later row up by one,
// and a new row always takes the next number.
//
// The number is derived, never stored, so the raw ids stay untouched.

// seqByID returns, for each given id, the row's 1-based position among the rows
// of table matching scope, ordered by ascending id.
//
// scope is an extra predicate on the inner alias (t2) that must describe the row
// set of the listing, so a row the listing hides cannot consume a number: pass
// "t2.deleted_at IS NULL" for tables with soft delete, and "" for tables where
// every stored row is listed.
//
// The correlated subquery only reads the primary key index and is portable
// across SQLite, MySQL and PostgreSQL.
func seqByID(db *gorm.DB, table string, scope string, ids []int) (map[int]int, error) {
	seqs := make(map[int]int, len(ids))
	if len(ids) == 0 {
		return seqs, nil
	}

	inner := "t2.id <= t.id"
	if scope != "" {
		inner += " AND " + scope
	}
	sql := fmt.Sprintf(
		"SELECT t.id AS id, (SELECT COUNT(*) FROM %s t2 WHERE %s) AS seq FROM %s t WHERE t.id IN (?)",
		table, inner, table,
	)

	var rows []struct {
		Id  int `gorm:"column:id"`
		Seq int `gorm:"column:seq"`
	}
	if err := db.Raw(sql, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		seqs[row.Id] = row.Seq
	}
	return seqs, nil
}

// attachSeq fills Seq on every row of a listing page. Rows without a stored id
// (derived rows that only exist in the response) keep the zero value.
func attachSeq[T any](db *gorm.DB, table string, scope string, rows []*T, idOf func(*T) int, setSeq func(*T, int)) error {
	if len(rows) == 0 {
		return nil
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, idOf(row))
	}

	seqs, err := seqByID(db, table, scope, ids)
	if err != nil {
		return err
	}

	for _, row := range rows {
		setSeq(row, seqs[idOf(row)])
	}
	return nil
}

// AttachUserSeq numbers users by creation order. The user listing is unscoped:
// soft-deleted users are still shown (flagged as deleted), so every stored row
// keeps its number and the column stays gapless.
func AttachUserSeq(db *gorm.DB, users []*User) error {
	return attachSeq(db, "users", "", users,
		func(user *User) int { return user.Id },
		func(user *User, seq int) { user.Seq = seq },
	)
}

// AttachRedemptionSeq numbers redemption codes by creation order. The listing
// hides soft-deleted codes, so they are excluded from the count and deleting one
// tightens the numbers of the codes below it.
func AttachRedemptionSeq(db *gorm.DB, redemptions []*Redemption) error {
	return attachSeq(db, "redemptions", "t2.deleted_at IS NULL", redemptions,
		func(redemption *Redemption) int { return redemption.Id },
		func(redemption *Redemption, seq int) { redemption.Seq = seq },
	)
}

// AttachModelSeq numbers model metadata by creation order. Derived rows that
// only exist because a channel serves a model have no stored id and keep Seq 0.
func AttachModelSeq(db *gorm.DB, models []*Model) error {
	return attachSeq(db, "models", "t2.deleted_at IS NULL", models,
		func(metadata *Model) int { return metadata.Id },
		func(metadata *Model, seq int) { metadata.Seq = seq },
	)
}

// AttachVendorSeq numbers vendors by creation order.
func AttachVendorSeq(db *gorm.DB, vendors []*Vendor) error {
	return attachSeq(db, "vendors", "t2.deleted_at IS NULL", vendors,
		func(vendor *Vendor) int { return vendor.Id },
		func(vendor *Vendor, seq int) { vendor.Seq = seq },
	)
}

// AttachUserSubscriptionSeq numbers subscriptions by creation order, scoped to
// the owning user: the console's subscription dialog numbers each user's
// subscriptions from 1 independently, so one user's rows never consume another
// user's numbers. Subscriptions are hard-deleted (no soft delete), so every
// stored row of the user is counted and deleting one tightens the rest.
func AttachUserSubscriptionSeq(db *gorm.DB, subs []UserSubscription) error {
	rows := make([]*UserSubscription, len(subs))
	for i := range subs {
		rows[i] = &subs[i]
	}
	return attachSeq(db, "user_subscriptions", "t2.user_id = t.user_id", rows,
		func(sub *UserSubscription) int { return sub.Id },
		func(sub *UserSubscription, seq int) { sub.Seq = seq },
	)
}

// AttachSubscriptionPlanSeq numbers subscription plans by creation order. Plans
// are hard-deleted (no soft delete) and the admin listing shows every stored
// plan, disabled ones included, so no scope is needed: deleting a plan tightens
// the numbers below it and a new plan always takes the next number.
func AttachSubscriptionPlanSeq(db *gorm.DB, plans []SubscriptionPlan) error {
	rows := make([]*SubscriptionPlan, len(plans))
	for i := range plans {
		rows[i] = &plans[i]
	}
	return attachSeq(db, "subscription_plans", "", rows,
		func(plan *SubscriptionPlan) int { return plan.Id },
		func(plan *SubscriptionPlan, seq int) { plan.Seq = seq },
	)
}
