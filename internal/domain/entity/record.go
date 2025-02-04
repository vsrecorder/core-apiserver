package entity

import (
	"time"
)

type Record struct {
	ID              string
	CreatedAt       time.Time
	OfficialEventId uint
	TonamelEventId  string
	FriendId        string
	UserId          string
	DeckId          string
	PrivateFlg      bool
	TCGMeisterURL   string
	Memo            string
}

func NewRecord(
	id string,
	createdAt time.Time,
	officialEventId uint,
	tonamelEventId string,
	friendId string,
	userId string,
	deckId string,
	privateFlg bool,
	tcgMeisterURL string,
	memo string,
) *Record {
	return &Record{
		ID:              id,
		CreatedAt:       createdAt,
		OfficialEventId: officialEventId,
		TonamelEventId:  tonamelEventId,
		FriendId:        friendId,
		UserId:          userId,
		DeckId:          deckId,
		PrivateFlg:      privateFlg,
		TCGMeisterURL:   tcgMeisterURL,
		Memo:            memo,
	}
}
