package model

import "time"

type PokemonAvatar struct {
	ID        int `gorm:"primaryKey"`
	Title     string
	ImageURL  string
	Detail    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewPokemonAvatar(
	id int,
	title string,
	imageURL string,
	detail string,
	createdAt time.Time,
	updatedAt time.Time,
) *PokemonAvatar {
	return &PokemonAvatar{
		ID:        id,
		Title:     title,
		ImageURL:  imageURL,
		Detail:    detail,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
