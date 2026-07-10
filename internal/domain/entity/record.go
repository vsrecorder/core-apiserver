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
	// DeckRegisteredAt は deck_id/deck_code_id が未設定→設定ありに変わった日時
	// (称号判定のasOf集計で使う。usecase.Record.Create/Updateが設定する)。
	// nil = 未設定。
	DeckRegisteredAt *time.Time
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
