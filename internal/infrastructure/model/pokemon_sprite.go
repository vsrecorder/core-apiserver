package model

type PokemonSprite struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func NewPokemonSprite(
	id string,
	name string,
) *PokemonSprite {
	return &PokemonSprite{
		ID:   id,
		Name: name,
	}
}
