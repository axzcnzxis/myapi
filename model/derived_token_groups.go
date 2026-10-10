package model

import (
	"fmt"

	"github.com/axzcnzxis/myapi/common"
	"gorm.io/gorm"
)

// DerivedTokenGroupsSyncer rewrites the group snapshot stored on a user's
// subscription-derived ("auto") API keys after the user's entitlement changed.
// It is implemented by the service package, which owns the derivation policy,
// and registered here so that model can drive it without importing service
// (service already imports model).
type DerivedTokenGroupsSyncer func(tx *gorm.DB, userId int, userGroup string, subscriptionGroups []string) error

var derivedTokenGroupsSyncer DerivedTokenGroupsSyncer

// RegisterDerivedTokenGroupsSyncer wires the service-side implementation. It is
// called from the service package's init; when nothing is registered (for
// example in model-only tests) the sync is simply skipped.
func RegisterDerivedTokenGroupsSyncer(syncer DerivedTokenGroupsSyncer) {
	derivedTokenGroupsSyncer = syncer
}

// syncDerivedTokenGroupsTx runs the registered syncer, if any, and only logs a
// failure: the surrounding transaction already persisted the authoritative
// entitlement, so aborting it would roll that back as well. A stale snapshot
// left behind is still corrected at request time by the service-side trim.
func syncDerivedTokenGroupsTx(tx *gorm.DB, userId int, userGroup string, subscriptionGroups []string) {
	if derivedTokenGroupsSyncer == nil {
		return
	}
	if err := derivedTokenGroupsSyncer(tx, userId, userGroup, subscriptionGroups); err != nil {
		common.SysError(fmt.Sprintf("failed to sync derived token groups for user %d: %v", userId, err))
	}
}
