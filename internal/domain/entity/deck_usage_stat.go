package entity

// DeckUsage は単一デッキの使用状況を表す
type DeckUsage struct {
	DeckId         string
	Name           string
	Count          int
	UsageRate      float64
	PokemonSprites []*PokemonSprite
}

func NewDeckUsage(
	deckId string,
	name string,
	count int,
	usageRate float64,
	pokemonSprites []*PokemonSprite,
) *DeckUsage {
	return &DeckUsage{
		DeckId:         deckId,
		Name:           name,
		Count:          count,
		UsageRate:      usageRate,
		PokemonSprites: pokemonSprites,
	}
}

// DeckUsageStat はユーザーのデッキ使用率分析の結果を表す
type DeckUsageStat struct {
	UserId       string
	TotalRecords int
	Decks        []*DeckUsage
}

func NewDeckUsageStat(
	userId string,
	totalRecords int,
	decks []*DeckUsage,
) *DeckUsageStat {
	return &DeckUsageStat{
		UserId:       userId,
		TotalRecords: totalRecords,
		Decks:        decks,
	}
}
