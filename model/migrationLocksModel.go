package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MigrationLocksModel = (*customMigrationLocksModel)(nil)

type (
	// MigrationLocksModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMigrationLocksModel.
	MigrationLocksModel interface {
		migrationLocksModel
	}

	customMigrationLocksModel struct {
		*defaultMigrationLocksModel
	}
)

// NewMigrationLocksModel returns a model for the database table.
func NewMigrationLocksModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) MigrationLocksModel {
	return &customMigrationLocksModel{
		defaultMigrationLocksModel: newMigrationLocksModel(conn, c, opts...),
	}
}
