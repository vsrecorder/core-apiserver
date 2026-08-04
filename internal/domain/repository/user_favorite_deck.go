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
	// デッキの削除・退会に伴う一括解除はここではなく DeckInterface 側
	// (Delete / DeleteByUserId)が担う。デッキ本体の削除と同じトランザクションで
	// 消さないと、参照先の無いお気に入りが残り得るため。
	Delete(
		ctx context.Context,
		uid string,
		deckId string,
	) error
}
