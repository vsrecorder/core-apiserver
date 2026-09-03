package model

import "time"

// ChampionsleagueResult は championsleague_results テーブルの1行。
// cityleague_results と違い point カラムを持たない(公式APIは返すがテーブルに列が無い)。
type ChampionsleagueResult struct {
	ChampionsleagueScheduleId string `gorm:"primaryKey"`
	OfficialEventId           uint   `gorm:"primaryKey"`
	LeagueType                uint
	EventDate                 time.Time
	PlayerId                  string `gorm:"primaryKey"`
	PlayerName                string
	Rank                      uint
	DeckCode                  string
}

func NewChampionsleagueResult(
	championsleagueScheduleId string,
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
	playerId string,
	playerName string,
	rank uint,
	deckCode string,
) *ChampionsleagueResult {
	return &ChampionsleagueResult{
		ChampionsleagueScheduleId: championsleagueScheduleId,
		OfficialEventId:           officialEventId,
		LeagueType:                leagueType,
		EventDate:                 eventDate,
		PlayerId:                  playerId,
		PlayerName:                playerName,
		Rank:                      rank,
		DeckCode:                  deckCode,
	}
}
