package model

import (
	"github.com/axzcnzxis/myapi/common"
	"gorm.io/gorm"
)

// GetDBTimestamp returns a UNIX timestamp from database time.
// Falls back to application time on error.
func GetDBTimestamp() int64 {
	return dbTimestamp(DB)
}

// GetDBTimestampTx reads the database clock through the provided handle.
// Callers that already hold a transaction must use this variant: reading through
// the global handle would need a second connection, which deadlocks a pool
// limited to a single connection (as in the model test harness).
func GetDBTimestampTx(tx *gorm.DB) int64 {
	if tx == nil {
		return common.GetTimestamp()
	}
	return dbTimestamp(tx)
}

func dbTimestamp(handle *gorm.DB) int64 {
	if handle == nil {
		return common.GetTimestamp()
	}
	var ts int64
	var err error
	switch {
	case common.UsingMainDatabase(common.DatabaseTypePostgreSQL):
		err = handle.Raw("SELECT EXTRACT(EPOCH FROM NOW())::bigint").Scan(&ts).Error
	case common.UsingMainDatabase(common.DatabaseTypeSQLite):
		err = handle.Raw("SELECT strftime('%s','now')").Scan(&ts).Error
	default:
		err = handle.Raw("SELECT UNIX_TIMESTAMP()").Scan(&ts).Error
	}
	if err != nil || ts <= 0 {
		return common.GetTimestamp()
	}
	return ts
}
