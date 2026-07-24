package model

// PokemonSprite は pokemon_sprites マスタ(スプライトID・正式名)。
type PokemonSprite struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

// DeckNameAlias はデッキ名エイリアス辞書の1行。
// スプライト未設定のマッチ/デッキをデッキ名から代表スプライトへ解決するために使う。
type DeckNameAlias struct {
	Alias           string `gorm:"primaryKey"`
	Position        uint   `gorm:"primaryKey"`
	PokemonSpriteId string
}

// TableName は gorm の複数形推測("deck_name_alias"→不定形)に任せず明示する。
func (DeckNameAlias) TableName() string { return "deck_name_aliases" }

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
