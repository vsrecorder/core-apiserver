package model

type MatchPokemonSprite struct {
	MatchId         string `gorm:"primaryKey"`
	Position        uint   `gorm:"primaryKey"`
	PokemonSpriteId string
}

func NewMatchPokemonSprite(
	match_id string,
	position uint,
	pokemon_sprite_id string,
) *MatchPokemonSprite {
	return &MatchPokemonSprite{
		MatchId:         match_id,
		Position:        position,
		PokemonSpriteId: pokemon_sprite_id,
	}
}

type DeckPokemonSprite struct {
	DeckId          string `gorm:"primaryKey"`
	Position        uint   `gorm:"primaryKey"`
	PokemonSpriteId string
}

func NewDeckPokemonSprite(
	deck_id string,
	position uint,
	pokemon_sprite_id string,
) *DeckPokemonSprite {
	return &DeckPokemonSprite{
		DeckId:          deck_id,
		Position:        position,
		PokemonSpriteId: pokemon_sprite_id,
	}
}
