package entity

import "time"

type RecentMatch struct {
	Sequence          int
	EventDate         time.Time
	DeckId            string
	OpponentsDeckInfo string
	VictoryFlg        bool
	RollingWinRate    float64
	EnvironmentId     string
	EnvironmentTitle  string
	PokemonSprites    []*PokemonSprite
}

func NewRecentMatch(
	sequence int,
	eventDate time.Time,
	deckId string,
	opponentsDeckInfo string,
	victoryFlg bool,
	rollingWinRate float64,
	environmentId string,
	environmentTitle string,
	pokemonSprites []*PokemonSprite,
) *RecentMatch {
	return &RecentMatch{
		Sequence:          sequence,
		EventDate:         eventDate,
		DeckId:            deckId,
		OpponentsDeckInfo: opponentsDeckInfo,
		VictoryFlg:        victoryFlg,
		RollingWinRate:    rollingWinRate,
		EnvironmentId:     environmentId,
		EnvironmentTitle:  environmentTitle,
		PokemonSprites:    pokemonSprites,
	}
}

type RecentMatchStat struct {
	UserId       string
	Count        int
	TotalMatches int
	Wins         int
	WinRate      float64
	Matches      []*RecentMatch
}

func NewRecentMatchStat(
	userId string,
	count int,
	totalMatches int,
	wins int,
	winRate float64,
	matches []*RecentMatch,
) *RecentMatchStat {
	return &RecentMatchStat{
		UserId:       userId,
		Count:        count,
		TotalMatches: totalMatches,
		Wins:         wins,
		WinRate:      winRate,
		Matches:      matches,
	}
}
