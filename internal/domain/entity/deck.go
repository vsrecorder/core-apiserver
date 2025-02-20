package entity

import "time"

type Deck struct {
	ID             string
	CreatedAt      time.Time
	ArchivedAt     time.Time
	UserId         string
	Name           string
	Code           string
	PrivateCodeFlg bool
}

func NewDeck(
	id string,
	createdAt time.Time,
	archivedAt time.Time,
	userId string,
	name string,
	code string,
	privateCodeFlg bool,
) *Deck {
	return &Deck{
		ID:             id,
		CreatedAt:      createdAt,
		ArchivedAt:     archivedAt,
		UserId:         userId,
		Name:           name,
		Code:           code,
		PrivateCodeFlg: privateCodeFlg,
	}
}
