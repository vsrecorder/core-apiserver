package entity

import "time"

type Deck struct {
	ID             string
	CreatedAt      time.Time
	ArchivedAt     time.Time
	FavoritedAt    time.Time
	UserId         string
	Name           string
	PrivateFlg     bool
	LatestDeckCode *DeckCode
	PokemonSprites []*PokemonSprite
}

func NewDeck(
	id string,
	createdAt time.Time,
	archivedAt time.Time,
	favoritedAt time.Time,
	userId string,
	name string,
	privateFlg bool,
	latestDeckCode *DeckCode,
	pokemonSprites []*PokemonSprite,
) *Deck {
	return &Deck{
		ID:             id,
		CreatedAt:      createdAt,
		ArchivedAt:     archivedAt,
		FavoritedAt:    favoritedAt,
		UserId:         userId,
		Name:           name,
		PrivateFlg:     privateFlg,
		LatestDeckCode: latestDeckCode,
		PokemonSprites: pokemonSprites,
	}
}
