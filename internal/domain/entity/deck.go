package entity

import "time"

type Deck struct {
	ID             string
	CreatedAt      time.Time
	ArchivedAt     time.Time
	UserId         string
	Name           string
	PrivateFlg     bool
	LatestDeckCode *DeckCode
}

func NewDeck(
	id string,
	createdAt time.Time,
	archivedAt time.Time,
	userId string,
	name string,
	privateFlg bool,
	latestDeckCode *DeckCode,
) *Deck {
	return &Deck{
		ID:             id,
		CreatedAt:      createdAt,
		ArchivedAt:     archivedAt,
		UserId:         userId,
		Name:           name,
		PrivateFlg:     privateFlg,
		LatestDeckCode: latestDeckCode,
	}
}
