package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TasksModel = (*customTasksModel)(nil)

type (
	// TasksModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTasksModel.
	TasksModel interface {
		tasksModel
	}

	customTasksModel struct {
		*defaultTasksModel
	}
)

// NewTasksModel returns a model for the database table.
func NewTasksModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TasksModel {
	return &customTasksModel{
		defaultTasksModel: newTasksModel(conn, c, opts...),
	}
}
