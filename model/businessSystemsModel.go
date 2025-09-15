package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ BusinessSystemsModel = (*customBusinessSystemsModel)(nil)

type (
	// BusinessSystemsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customBusinessSystemsModel.
	BusinessSystemsModel interface {
		businessSystemsModel
	}

	customBusinessSystemsModel struct {
		*defaultBusinessSystemsModel
	}
)

// NewBusinessSystemsModel returns a model for the database table.
func NewBusinessSystemsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) BusinessSystemsModel {
	return &customBusinessSystemsModel{
		defaultBusinessSystemsModel: newBusinessSystemsModel(conn, c, opts...),
	}
}
