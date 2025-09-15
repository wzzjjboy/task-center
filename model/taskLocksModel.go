package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TaskLocksModel = (*customTaskLocksModel)(nil)

type (
	// TaskLocksModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTaskLocksModel.
	TaskLocksModel interface {
		taskLocksModel
	}

	customTaskLocksModel struct {
		*defaultTaskLocksModel
	}
)

// NewTaskLocksModel returns a model for the database table.
func NewTaskLocksModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TaskLocksModel {
	return &customTaskLocksModel{
		defaultTaskLocksModel: newTaskLocksModel(conn, c, opts...),
	}
}
