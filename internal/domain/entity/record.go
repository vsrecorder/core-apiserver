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
	IgnoreStatsFlg    bool
	// RegulationId は regulations テーブルのID。未指定(0)は usecase 層で
	// 既定のスタンダードへ寄せるため、保存時は常に1以上になる。
	RegulationId  uint
	TCGMeisterURL string
	Memo          string
	// DeckRegisteredAt は deck_id/deck_code_id が未設定→設定ありに変わった日時
	// (称号判定のasOf集計で使う。usecase.Record.Create/Updateが設定する)。
	// nil = 未設定。
	DeckRegisteredAt *time.Time
	// Tags は付与されたタグ。読み込み時にインフラ層が詰める。
	// 付与の書き込みは TagRepository.ReplaceRecordTags が担うため、
	// NewRecord のコンストラクタ引数には含めない(deck / match の Tags と同じ扱い)。
	Tags []*Tag
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
	ignoreStatsFlg bool,
	regulationId uint,
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
		IgnoreStatsFlg:    ignoreStatsFlg,
		RegulationId:      regulationId,
		TCGMeisterURL:     tcgMeisterURL,
		Memo:              memo,
	}
}
