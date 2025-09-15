package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TaskExecutionsModel = (*customTaskExecutionsModel)(nil)

type (
	// TaskExecutionsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTaskExecutionsModel.
	TaskExecutionsModel interface {
		taskExecutionsModel
	}

	customTaskExecutionsModel struct {
		*defaultTaskExecutionsModel
	}
)

// NewTaskExecutionsModel returns a model for the database table.
func NewTaskExecutionsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TaskExecutionsModel {
	return &customTaskExecutionsModel{
		defaultTaskExecutionsModel: newTaskExecutionsModel(conn, c, opts...),
	}
}
