package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type DeckCodeInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.DeckCode, error)

	FindByDeckId(
		ctx context.Context,
		deckId string,
	) ([]*entity.DeckCode, error)

	// FindIdsByUserId は退会時の連鎖削除など、ID一覧だけを軽量に取得したい場合に使う。
	// DeckCode.DeckId は必ずしもそのユーザ自身が所有するデッキとは限らないため、
	// Deck の連鎖削除とは別に user_id で直接洗い出す必要がある。
	FindIdsByUserId(
		ctx context.Context,
		uid string,
	) ([]string, error)

	Save(
		ctx context.Context,
		entity *entity.DeckCode,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
