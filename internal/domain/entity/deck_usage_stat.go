package entity

// DeckUsage は単一デッキの使用状況を表す
type DeckUsage struct {
	DeckId    string
	Name      string
	Count     int
	UsageRate float64
	Wins      int
	Losses    int
	WinRate   float64
	// GameCount は先攻/後攻が記録されたゲーム数（BO3の場合は1対戦で複数ゲーム）
	GameCount       int
	GoFirstCount    int
	GoSecondCount   int
	GoFirstRate     float64
	GoFirstWins     int
	GoFirstWinRate  float64
	GoSecondWins    int
	GoSecondWinRate float64
	PokemonSprites  []*PokemonSprite
}

func NewDeckUsage(
	deckId string,
	name string,
	count int,
	usageRate float64,
	wins int,
	losses int,
	winRate float64,
	gameCount int,
	goFirstCount int,
	goSecondCount int,
	goFirstRate float64,
	goFirstWins int,
	goFirstWinRate float64,
	goSecondWins int,
	goSecondWinRate float64,
	pokemonSprites []*PokemonSprite,
) *DeckUsage {
	return &DeckUsage{
		DeckId:          deckId,
		Name:            name,
		Count:           count,
		UsageRate:       usageRate,
		Wins:            wins,
		Losses:          losses,
		WinRate:         winRate,
		GameCount:       gameCount,
		GoFirstCount:    goFirstCount,
		GoSecondCount:   goSecondCount,
		GoFirstRate:     goFirstRate,
		GoFirstWins:     goFirstWins,
		GoFirstWinRate:  goFirstWinRate,
		GoSecondWins:    goSecondWins,
		GoSecondWinRate: goSecondWinRate,
		PokemonSprites:  pokemonSprites,
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
