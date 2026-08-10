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
	// Tags は付与されたタグ。読み込み時にインフラ層が詰める。
	// 付与の書き込みは Deck.Save ではなく TagRepository.ReplaceDeckTags が担うため、
	// コンストラクタ引数には含めない(アーカイブ等の別経路で誤って空へ上書きしないため)。
	Tags []*Tag
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
