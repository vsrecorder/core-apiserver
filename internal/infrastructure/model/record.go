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
	// レギュレーション。未設定(ゼロ値)のときはDB側の DEFAULT(スタンダード)に任せる。
	// records.regulation_id は regulations へのFK制約付きで、0のままでは保存できない。
	RegulationId  uint `gorm:"default:1"`
	TCGMeisterURL string
	Memo          string
	// 自由形式イベント用。開催日(EventDate)はユーザ入力値を保持し、
	// イベント本体は unofficial_events テーブルへ分離して UnofficialEventId で参照する。
	EventDate         time.Time
	UnofficialEventId string
	// DeckRegisteredAt は deck_id/deck_code_id が未設定→設定ありに変わった日時
	// (称号判定のasOf集計で使う。usecase.Record.Create/Updateが設定する)。
	// time.Time(非ポインタ)だとゼロ値がSaveのたびにNULLではなく'0001-01-01'として
	// 書き込まれてしまうため、未設定=nilを表現できるポインタ型にする。
	DeckRegisteredAt *time.Time
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
	regulationId uint,
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
		RegulationId:      regulationId,
		TCGMeisterURL:     tcgMeisterURL,
		Memo:              memo,
		EventDate:         eventDate,
		UnofficialEventId: unofficialEventId,
	}
}
