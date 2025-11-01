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
