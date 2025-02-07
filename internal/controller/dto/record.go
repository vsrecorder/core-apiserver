package dto

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type RecordCreateRequest struct {
	OfficialEventId uint   `json:"official_event_id"`
	TonamelEventId  string `json:"tonamel_event_id"`
	FriendId        string `json:"friend_id"`
	DeckId          string `json:"deck_id"`
	PrivateFlg      bool   `json:"private_flg"`
	TCGMeisterURL   string `json:"tcg_meister_url"`
	Memo            string `json:"memo"`
}

type RecordUpdateRequest struct {
	OfficialEventId uint   `json:"official_event_id"`
	TonamelEventId  string `json:"tonamel_event_id"`
	FriendId        string `json:"friend_id"`
	DeckId          string `json:"deck_id"`
	PrivateFlg      bool   `json:"private_flg"`
	TCGMeisterURL   string `json:"tcg_meister_url"`
	Memo            string `json:"memo"`
}

type RecordGetResponse struct {
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
	Records []*entity.Record `json:"records"`
}

type RecordCreateResponse struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	OfficialEventId uint      `json:"official_event_id"`
	TonamelEventId  string    `json:"tonamel_event_id"`
	FriendId        string    `json:"friend_id"`
	UserId          string    `json:"user_id"`
	DeckId          string    `json:"deck_id"`
	PrivateFlg      bool      `json:"private_flg"`
	TCGMeisterURL   string    `json:"tcg_meister_url"`
	Memo            string    `json:"memo"`
}

type RecordUpdateResponse struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	OfficialEventId uint      `json:"official_event_id"`
	TonamelEventId  string    `json:"tonamel_event_id"`
	FriendId        string    `json:"friend_id"`
	UserId          string    `json:"user_id"`
	DeckId          string    `json:"deck_id"`
	PrivateFlg      bool      `json:"private_flg"`
	TCGMeisterURL   string    `json:"tcg_meister_url"`
	Memo            string    `json:"memo"`
}
