package dto

import (
	"time"
)

type RecordRequest struct {
	OfficialEventId uint   `json:"official_event_id"`
	TonamelEventId  string `json:"tonamel_event_id"`
	FriendId        string `json:"friend_id"`
	DeckId          string `json:"deck_id"`
	PrivateFlg      bool   `json:"private_flg"`
	TCGMeisterURL   string `json:"tcg_meister_url"`
	Memo            string `json:"memo"`
}

type RecordCreateRequest struct {
	RecordRequest
}

type RecordUpdateRequest struct {
	RecordRequest
}

type RecordResponse struct {
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

type RecordGetResponse struct {
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	Records []*RecordResponse `json:"records"`
}

type RecordGetByIdResponse struct {
	RecordResponse
}

type RecordCreateResponse struct {
	RecordResponse
}

type RecordUpdateResponse struct {
	RecordResponse
}
