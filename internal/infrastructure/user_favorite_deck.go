package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UserFavoriteDeck struct {
	db *gorm.DB
}

func NewUserFavoriteDeck(db *gorm.DB) repository.UserFavoriteDeckInterface {
	return &UserFavoriteDeck{db}
}

func (i *UserFavoriteDeck) FindByUserId(
	ctx context.Context,
	uid string,
) ([]*entity.UserFavoriteDeck, error) {
	db := dbFromContext(ctx, i.db)

	var favorites []*model.UserFavoriteDeck

	// 論理削除済みのデッキへのお気に入りは、一覧にも件数にも数えない。
	// (デッキを消しただけで枠が埋まったままになるのを防ぐ)
	if tx := db.Table(
		"user_favorite_decks",
	).Select(
		"user_favorite_decks.user_id, user_favorite_decks.deck_id, user_favorite_decks.created_at",
	).Joins(
		"JOIN decks ON decks.id = user_favorite_decks.deck_id AND decks.deleted_at IS NULL",
	).Where(
		"user_favorite_decks.user_id = ?", uid,
	).Order(
		"user_favorite_decks.created_at ASC",
	).Scan(&favorites); tx.Error != nil {
		return nil, tx.Error
	}

	ret := make([]*entity.UserFavoriteDeck, 0, len(favorites))
	for _, favorite := range favorites {
		ret = append(ret, entity.NewUserFavoriteDeck(
			favorite.UserId,
			favorite.DeckId,
			favorite.CreatedAt,
		))
	}

	return ret, nil
}

func (i *UserFavoriteDeck) Create(
	ctx context.Context,
	entity *entity.UserFavoriteDeck,
) error {
	db := dbFromContext(ctx, i.db)

	favorite := model.NewUserFavoriteDeck(
		entity.UserId,
		entity.DeckId,
		entity.CreatedAt,
	)

	return db.Create(favorite).Error
}

func (i *UserFavoriteDeck) Delete(
	ctx context.Context,
	uid string,
	deckId string,
) error {
	db := dbFromContext(ctx, i.db)

	return db.Where(
		"user_id = ? AND deck_id = ?", uid, deckId,
	).Delete(&model.UserFavoriteDeck{}).Error
}
