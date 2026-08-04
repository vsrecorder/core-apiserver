package entity

import "time"

// UserFavoriteDeck は「あるユーザがあるデッキをお気に入りにしている」ことを表す。
// CreatedAt はお気に入りにした日時で、上限に達したときにどれを外すか(古い順)の判断に使う。
type UserFavoriteDeck struct {
	UserId    string
	DeckId    string
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
