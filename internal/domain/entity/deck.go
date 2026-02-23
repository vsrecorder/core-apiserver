package entity

import "time"

type Deck struct {
	ID             string
	CreatedAt      time.Time
	ArchivedAt     time.Time
	UserId         string
	Name           string
	Code           string
	PrivateFlg     bool
	LatestDeckCode *DeckCode
}

func NewDeck(
	id string,
	createdAt time.Time,
	archivedAt time.Time,
	userId string,
	name string,
	code string,
	privateFlg bool,
	latestDeckCode *DeckCode,
) *Deck {
	return &Deck{
		ID:             id,
		CreatedAt:      createdAt,
		ArchivedAt:     archivedAt,
		UserId:         userId,
		Name:           name,
		Code:           code,
		PrivateFlg:     privateFlg,
		LatestDeckCode: latestDeckCode,
	}
}
