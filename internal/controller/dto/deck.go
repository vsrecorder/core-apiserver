package dto

import "time"

type DeckCreateRequest struct {
	Name               string `json:"name"`
	PrivateFlg         bool   `json:"private_flg"`
	DeckCode           string `json:"deck_code"`
	PrivateDeckCodeFlg bool   `json:"private_deck_code_flg"`
}

type DeckUpdateRequest struct {
	Name       string `json:"name"`
	PrivateFlg bool   `json:"private_flg"`
}

type DeckData struct {
	Cursor string        `json:"cursor"`
	Data   *DeckResponse `json:"data"`
}

type DeckResponse struct {
	ID             string           `json:"id"`
	CreatedAt      time.Time        `json:"created_at"`
	ArchivedAt     time.Time        `json:"archived_at"`
	UserId         string           `json:"user_id"`
	Name           string           `json:"name"`
	PrivateFlg     bool             `json:"private_flg"`
	LatestDeckCode DeckCodeResponse `json:"latest_deck_code"`
}

type DeckGetResponse struct {
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Cursor string      `json:"cursor"`
	Decks  []*DeckData `json:"decks"`
}

type DeckGetAllResponse []DeckResponse

type DeckGetByIdResponse struct {
	DeckResponse
}

type DeckGetByUserIdResponse struct {
	Archived bool        `json:"archived"`
	Limit    int         `json:"limit"`
	Offset   int         `json:"offset"`
	Cursor   string      `json:"cursor"`
	Decks    []*DeckData `json:"decks"`
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
