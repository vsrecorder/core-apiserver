package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type KizunaInterface interface {
	// FindKizunaDeckAggregates はユーザーの全デッキぶんの、きずなLv.算出に必要な
	// 集計済みの素の値を返す。記録が1件も無いデッキも（ゼロ値で）含める。
	// 点数付けは entity.CalculateKizuna が行うため、ここでは事実の集計だけを返す。
	FindKizunaDeckAggregates(
		ctx context.Context,
		userId string,
	) ([]*entity.KizunaDeckAggregate, error)
}
