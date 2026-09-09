package entity

import "time"

type RecentMatch struct {
	Sequence  int
	EventDate time.Time
	// OfficialEventId は紐づく公式イベント(無ければ0)。開催日と実際の対戦環境がズレる
	// イベントの環境を引き当てるために持つ(表示には使わない)。
	OfficialEventId   uint
	DeckId            string
	OpponentsDeckInfo string
	VictoryFlg        bool
	DrawFlg           bool
	RollingWinRate    float64
	EnvironmentId     string
	EnvironmentTitle  string
	PokemonSprites    []*PokemonSprite
}

func NewRecentMatch(
	sequence int,
	eventDate time.Time,
	officialEventId uint,
	deckId string,
	opponentsDeckInfo string,
	victoryFlg bool,
	drawFlg bool,
	rollingWinRate float64,
	environmentId string,
	environmentTitle string,
	pokemonSprites []*PokemonSprite,
) *RecentMatch {
	return &RecentMatch{
		Sequence:          sequence,
		EventDate:         eventDate,
		OfficialEventId:   officialEventId,
		DeckId:            deckId,
		OpponentsDeckInfo: opponentsDeckInfo,
		VictoryFlg:        victoryFlg,
		DrawFlg:           drawFlg,
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
