package entity

import (
	"time"
)

type EventResult struct {
	PlayerId   string
	PlayerName string
	Rank       uint
	Point      uint
	DeckCode   string
}

func NewEventResult(
	playerId string,
	playerName string,
	rank uint,
	point uint,
	deckCode string,
) *EventResult {
	return &EventResult{
		PlayerId:   playerId,
		PlayerName: playerName,
		Rank:       rank,
		Point:      point,
		DeckCode:   deckCode,
	}
}

// CityleagueResultEvent は結果が登録されているイベントを、入賞者を含めずに表す。
// sitemap の生成のように、イベントの識別子と開催日だけを必要とする用途で使う。
type CityleagueResultEvent struct {
	OfficialEventId uint
	LeagueType      uint
	EventDate       time.Time
}

func NewCityleagueResultEvent(
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
) *CityleagueResultEvent {
	return &CityleagueResultEvent{
		OfficialEventId: officialEventId,
		LeagueType:      leagueType,
		EventDate:       eventDate,
	}
}

type CityleagueResult struct {
	CityleagueScheduleId string
	OfficialEventId      uint
	LeagueType           uint
	EventDate            time.Time
	EventResults         []*EventResult
}

func NewCityleagueResult(
	cityleagueScheduleId string,
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
	eventResults []*EventResult,
) *CityleagueResult {
	return &CityleagueResult{
		CityleagueScheduleId: cityleagueScheduleId,
		OfficialEventId:      officialEventId,
		LeagueType:           leagueType,
		EventDate:            eventDate,
		EventResults:         eventResults,
	}
}
