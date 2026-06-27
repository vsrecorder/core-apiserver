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
}
