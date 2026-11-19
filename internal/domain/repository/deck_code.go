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

	// DeleteByUserId は退会時に、そのユーザが作成したデッキコードをまとめて論理削除する。
	// DeckCode.DeckId は必ずしもそのユーザ自身が所有するデッキとは限らないため、
	// Deck の連鎖削除とは別に user_id で直接消す必要がある。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error

	Save(
		ctx context.Context,
		entity *entity.DeckCode,
	) error

	// ReplacePokemonSprites は deckCodeId のスプライトを sprites の内容に置き換える(空なら未設定に戻す)。
	// デッキコード本体(Save)とは別テーブルのため分けてある。本体と食い違わないよう、
	// 呼び出し側は同じトランザクションの中で呼ぶこと。
	// 存在しないスプライトIDは apperror.ErrInvalidReference で返る。
	ReplacePokemonSprites(
		ctx context.Context,
		deckCodeId string,
		sprites []*entity.PokemonSprite,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
