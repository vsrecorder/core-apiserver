package entity

import (
	"time"
)

type Record struct {
	ID                string
	CreatedAt         time.Time
	OfficialEventId   uint
	TonamelEventId    string
	FriendId          string
	UnofficialEventId string
	UserId            string
	DeckId            string
	DeckCodeId        string
	EventDate         time.Time
	PrivateFlg        bool
	TCGMeisterURL     string
	Memo              string
}

func NewRecord(
	id string,
	createdAt time.Time,
	officialEventId uint,
	tonamelEventId string,
	friendId string,
	unofficialEventId string,
	userId string,
	deckId string,
	deckCodeId string,
	eventDate time.Time,
	privateFlg bool,
	tcgMeisterURL string,
	memo string,
) *Record {
	return &Record{
		ID:                id,
		CreatedAt:         createdAt,
		OfficialEventId:   officialEventId,
		TonamelEventId:    tonamelEventId,
		UnofficialEventId: unofficialEventId,
		FriendId:          friendId,
		UserId:            userId,
		DeckId:            deckId,
		DeckCodeId:        deckCodeId,
		EventDate:         eventDate,
		PrivateFlg:        privateFlg,
		TCGMeisterURL:     tcgMeisterURL,
		Memo:              memo,
	}
}
