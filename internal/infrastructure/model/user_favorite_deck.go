package model

import "time"

// UserFavoriteDeck は「あるユーザがあるデッキをお気に入りにしている」ことを表す。
//
// decks の列ではなく別テーブルなのは、1ユーザが持てる件数を将来増やせるようにするため。
// 解除は行の削除で表すため、論理削除(DeletedAt)は持たない。
type UserFavoriteDeck struct {
	UserId    string `gorm:"primaryKey"`
	DeckId    string `gorm:"primaryKey"`
	CreatedAt time.Time
}

func NewUserFavoriteDeck(
	userId string,
	deckId string,
	createdAt time.Time,
) *UserFavoriteDeck {
	return &UserFavoriteDeck{
		UserId:    userId,
		DeckId:    deckId,
		CreatedAt: createdAt,
	}
}
