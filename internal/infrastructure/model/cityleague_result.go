package model

import "time"

// CityleagueResult は cityleague_results テーブルの1行。
// プレイヤー名は持たない(players テーブルが持つ。model.Player 参照)。
type CityleagueResult struct {
	CityleagueScheduleId string `gorm:"primaryKey"`
	OfficialEventId      uint   `gorm:"primaryKey"`
	LeagueType           uint
	EventDate            time.Time
	PlayerId             string `gorm:"primaryKey"`
	Rank                 uint
	Point                uint
	DeckCode             string
}

func NewCityleagueResult(
	cityleagueScheduleId string,
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
	playerId string,
	rank uint,
	point uint,
	deckCode string,
) *CityleagueResult {
	return &CityleagueResult{
		CityleagueScheduleId: cityleagueScheduleId,
		OfficialEventId:      officialEventId,
		LeagueType:           leagueType,
		EventDate:            eventDate,
		PlayerId:             playerId,
		Rank:                 rank,
		Point:                point,
		DeckCode:             deckCode,
	}
}
