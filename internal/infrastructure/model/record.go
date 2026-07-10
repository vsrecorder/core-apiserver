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
	IgnoreStatsFlg  bool
	TCGMeisterURL   string
	Memo            string
	// 自由形式イベント用。開催日(EventDate)はユーザ入力値を保持し、
	// イベント本体は unofficial_events テーブルへ分離して UnofficialEventId で参照する。
	EventDate         time.Time
	UnofficialEventId string
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
	ignoreStatsFlg bool,
	tcgMeisterURL string,
	memo string,
	eventDate time.Time,
	unofficialEventId string,
) *Record {
	return &Record{
		ID:                id,
		CreatedAt:         createdAt,
		OfficialEventId:   officialEventId,
		TonamelEventId:    tonamelEventId,
		FriendId:          friendId,
		UserId:            userId,
		DeckId:            deckId,
		DeckCodeId:        deckCodeId,
		PrivateFlg:        privateFlg,
		IgnoreStatsFlg:    ignoreStatsFlg,
		TCGMeisterURL:     tcgMeisterURL,
		Memo:              memo,
		EventDate:         eventDate,
		UnofficialEventId: unofficialEventId,
	}
}
