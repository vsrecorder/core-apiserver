package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type RecordInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursor time.Time,
	) ([]*entity.Record, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.Record, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		limit int,
		cursor time.Time,
	) ([]*entity.Record, error)

	FindByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByTonamelEventId(
		ctx context.Context,
		tonamelEventId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByDeckId(
		ctx context.Context,
		deckId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByDeckCodeId(
		ctx context.Context,
		deckCodeId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	Save(
		ctx context.Context,
		entity *entity.Record,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
