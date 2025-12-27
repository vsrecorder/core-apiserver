package dto

import "time"

type DeckCodeCreateRequest struct {
	DeckId         string `json:"deck_id"`
	Code           string `json:"code"`
	PrivateCodeFlg bool   `json:"private_code_flg"`
}

type DeckCodeUpdateRequest struct {
	PrivateCodeFlg bool `json:"private_code_flg"`
}

type DeckCodeResponse struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserId         string    `json:"user_id"`
	DeckId         string    `json:"deck_id"`
	Code           string    `json:"code"`
	PrivateCodeFlg bool      `json:"private_code_flg"`
}

type DeckCodeGetByIdResponse struct {
	DeckCodeResponse
}

type DeckCodeCreateResponse struct {
	DeckCodeResponse
}

type DeckCodeUpdateResponse struct {
	DeckCodeResponse
}
