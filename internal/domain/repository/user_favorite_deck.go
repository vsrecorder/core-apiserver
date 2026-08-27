package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserFavoriteDeckInterface interface {
	// FindByUserId は uid のお気に入りを、お気に入りにした順(古い順)で返す。
	// 上限に達したときに最も古いものから外すため、並び順を保証する。
	FindByUserId(
		ctx context.Context,
		uid string,
	) ([]*entity.UserFavoriteDeck, error)

	// Create はお気に入りを1件追加する。
	Create(
		ctx context.Context,
		entity *entity.UserFavoriteDeck,
	) error

	// Delete は uid が deckId のデッキにつけたお気に入りを解除する。
	//
	// 「消えるデッキに付いていたお気に入り」の一括解除はここではなく DeckInterface 側
	// (Delete / DeleteByUserId)が担う。デッキ本体の削除と同じトランザクションで
	// 消さないと、参照先の無いお気に入りが残り得るため。
	Delete(
		ctx context.Context,
		uid string,
		deckId string,
	) error

	// DeleteByUserId は退会時に、そのユーザが付けたお気に入りをまとめて削除する。
	// DeckInterface 側の一括解除は「そのユーザのデッキに付いたお気に入り」が対象で、
	// 他人のデッキに付けたお気に入りは拾えないため、user_id 側からも消す必要がある。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
