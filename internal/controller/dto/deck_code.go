package dto

import "time"

type DeckCodeCreateRequest struct {
	DeckId         string `json:"deck_id"`
	Code           string `json:"code"`
	PrivateCodeFlg bool   `json:"private_code_flg"`
	Memo           string `json:"memo"`
}

type DeckCodeUpdateRequest struct {
	PrivateCodeFlg bool   `json:"private_code_flg"`
	Memo           string `json:"memo"`
}

type DeckCodeResponse struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserId         string    `json:"user_id"`
	DeckId         string    `json:"deck_id"`
	Code           string    `json:"code"`
	PrivateCodeFlg bool      `json:"private_code_flg"`
	Memo           string    `json:"memo"`
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
