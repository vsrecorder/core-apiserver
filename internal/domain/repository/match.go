package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type MatchInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Match, error)

	FindByRecordId(
		ctx context.Context,
		recordId string,
	) ([]*entity.Match, error)

	FindByUserId(
		ctx context.Context,
		userId string,
		limit int,
	) ([]*entity.Match, error)

	// FindSummariesByRecordIds は recordIds のうち userId が所有する記録について、
	// 対戦の集計を返す。他人の記録・存在しない記録はエラーにせず結果から除外する。
	// 対戦が1件も無い自分の記録は total=0 の集計として含める。
	FindSummariesByRecordIds(
		ctx context.Context,
		userId string,
		recordIds []string,
	) ([]*entity.MatchSummary, error)

	FindLatest(
		ctx context.Context,
		limit int,
	) ([]*entity.Match, error)

	Create(
		ctx context.Context,
		entity *entity.Match,
	) error

	Update(
		ctx context.Context,
		entity *entity.Match,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error

	Reorder(
		ctx context.Context,
		recordId string,
		orders []*entity.MatchOrder,
	) error
}
