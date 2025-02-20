package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type DeckInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*entity.Deck, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		archivedFlg bool,
		limit int,
		offset int,
	) ([]*entity.Deck, error)

	Save(
		ctx context.Context,
		entity *entity.Deck,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
