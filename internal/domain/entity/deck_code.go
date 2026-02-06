package entity

import "time"

type DeckCode struct {
	ID             string
	CreatedAt      time.Time
	UserId         string
	DeckId         string
	Code           string
	PrivateCodeFlg bool
	Memo           string
}

func NewDeckCode(
	id string,
	createdAt time.Time,
	userId string,
	deckId string,
	code string,
	privateCodeFlg bool,
	memo string,
) *DeckCode {
	return &DeckCode{
		ID:             id,
		CreatedAt:      createdAt,
		UserId:         userId,
		DeckId:         deckId,
		Code:           code,
		PrivateCodeFlg: privateCodeFlg,
		Memo:           memo,
	}
}
