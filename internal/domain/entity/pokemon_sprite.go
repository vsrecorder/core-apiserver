package entity

type PokemonSprite struct {
	ID string
}

func NewPokemonSprite(
	id string,
) *PokemonSprite {
	return &PokemonSprite{
		ID: id,
	}
}
