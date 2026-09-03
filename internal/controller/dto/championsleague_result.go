package dto

import "time"

// ChampionsleagueResultResponse は入賞1件。シティリーグ(ResultResponse)と違い point を持たない。
type ChampionsleagueResultResponse struct {
	PlayerId   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	Rank       uint   `json:"rank"`
	DeckCode   string `json:"deck_code"`
}

type ChampionsleagueEventResultResponse struct {
	ChampionsleagueScheduleId string                           `json:"championsleague_schedule_id"`
	OfficialEventId           uint                             `json:"official_event_id"`
	LeagueType                uint                             `json:"league_type"`
	Date                      time.Time                        `json:"date"`
	EventDetailResultURL      string                           `json:"event_detail_result_url"`
	Results                   []*ChampionsleagueResultResponse `json:"results"`
}

// ChampionsleagueResultEventResponse は入賞者を含まない、イベント単位の応答。
type ChampionsleagueResultEventResponse struct {
	ChampionsleagueScheduleId string    `json:"championsleague_schedule_id"`
	OfficialEventId           uint      `json:"official_event_id"`
	LeagueType                uint      `json:"league_type"`
	Date                      time.Time `json:"date"`
}

type ChampionsleagueResultGetEventsResponse struct {
	Count  int                                   `json:"count"`
	Events []*ChampionsleagueResultEventResponse `json:"events"`
}

type ChampionsleagueResultGetByChampionsleagueScheduleIdResponse struct {
	ChampionsleagueScheduleId string `json:"championsleague_schedule_id"`
	// 絞り込みに使ったリーグ区分。0 は全区分。
	LeagueType   uint                                  `json:"league_type"`
	Count        int                                   `json:"count"`
	EventResults []*ChampionsleagueEventResultResponse `json:"event_results"`
}
