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
