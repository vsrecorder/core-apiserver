package entity

// OpponentDeckUsage は単一の対戦相手デッキの対戦状況を表す
type OpponentDeckUsage struct {
	DeckInfo       string
	Count          int
	UsageRate      float64
	Wins           int
	Losses         int
	WinRate        float64
	PokemonSprites []*PokemonSprite
}

func NewOpponentDeckUsage(
	deckInfo string,
	count int,
	usageRate float64,
	wins int,
	losses int,
	winRate float64,
	pokemonSprites []*PokemonSprite,
) *OpponentDeckUsage {
	return &OpponentDeckUsage{
		DeckInfo:       deckInfo,
		Count:          count,
		UsageRate:      usageRate,
		Wins:           wins,
		Losses:         losses,
		WinRate:        winRate,
		PokemonSprites: pokemonSprites,
	}
}

// OpponentDeckUsageStat はユーザーの対戦相手デッキ分布の集計結果を表す
type OpponentDeckUsageStat struct {
	UserId       string
	TotalMatches int
	Decks        []*OpponentDeckUsage
}

func NewOpponentDeckUsageStat(
	userId string,
	totalMatches int,
	decks []*OpponentDeckUsage,
) *OpponentDeckUsageStat {
	return &OpponentDeckUsageStat{
		UserId:       userId,
		TotalMatches: totalMatches,
		Decks:        decks,
	}
}
