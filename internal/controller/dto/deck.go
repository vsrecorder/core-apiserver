package dto

import "time"

type DeckRequest struct {
	Name           string `json:"name"`
	Code           string `json:"code"`
	PrivateCodeFlg bool   `json:"private_code_flg"`
}

type DeckCreateRequest struct {
	DeckRequest
}

type DeckUpdateRequest struct {
	DeckRequest
}

type DeckResponse struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	ArchivedAt     time.Time `json:"archived_at"`
	UserId         string    `json:"user_id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	PrivateCodeFlg bool      `json:"private_code_flg"`
}

type DeckGetResponse struct {
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Decks  []*DeckResponse `json:"decks"`
}

type DeckGetByIdResponse struct {
	DeckResponse
}

type DeckGetByUserIdResponse struct {
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Decks  []*DeckResponse `json:"decks"`
}

type DeckCreateResponse struct {
	DeckResponse
}

type DeckUpdateResponse struct {
	DeckResponse
}

type DeckArchiveResponse struct {
	DeckResponse
}

type DeckUnarchiveResponse struct {
	DeckResponse
}
