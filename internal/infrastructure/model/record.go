package model

import (
	"time"

	"gorm.io/gorm"
)

type Record struct {
	ID              string `gorm:"primaryKey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
	OfficialEventId uint
	TonamelEventId  string
	FriendId        string
	UserId          string
	DeckId          string
	DeckCodeId      string
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
	deckCodeId string,
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
		DeckCodeId:      deckCodeId,
		PrivateFlg:      privateFlg,
		TCGMeisterURL:   tcgMeisterURL,
		Memo:            memo,
	}
}
