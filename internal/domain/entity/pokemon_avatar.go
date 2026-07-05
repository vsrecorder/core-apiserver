package entity

type PokemonAvatar struct {
	ID       int
	Title    string
	ImageURL string
	Detail   string
}

func NewPokemonAvatar(
	id int,
	title string,
	imageURL string,
	detail string,
) *PokemonAvatar {
	return &PokemonAvatar{
		ID:       id,
		Title:    title,
		ImageURL: imageURL,
		Detail:   detail,
	}
}
