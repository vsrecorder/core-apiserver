package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type DeckInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*entity.Deck, error)

	FindAll(
		ctx context.Context,
		uid string,
	) ([]*entity.Deck, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursor time.Time,
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

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		archivedFlg bool,
		limit int,
		cursor time.Time,
	) ([]*entity.Deck, error)

	// FindIdsByUserId は退会時の連鎖削除など、ID一覧だけを軽量に取得したい場合に使う。
	// (アーカイブ済みデッキも含めて全件を対象にする)
	FindIdsByUserId(
		ctx context.Context,
		uid string,
	) ([]string, error)

	Save(
		ctx context.Context,
		entity *entity.Deck,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
