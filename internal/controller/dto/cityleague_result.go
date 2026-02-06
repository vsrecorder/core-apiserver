package dto

import "time"

type ResultResponse struct {
	PlayerId   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	Rank       uint   `json:"rank"`
	Point      uint   `json:"point"`
	DeckCode   string `json:"deck_code"`
}

type EventResultResponse struct {
	CityleagueScheduleId string            `json:"cityleague_schedule_id"`
	OfficialEventId      uint              `json:"official_event_id"`
	LeagueType           uint              `json:"league_type"`
	Date                 time.Time         `json:"date"`
	EventDetailResultURL string            `json:"event_detail_result_url"`
	Results              []*ResultResponse `json:"results"`
}

type CityleagueResultGetResponse struct {
	LeagueType   uint                   `json:"league_type"`
	FromDate     time.Time              `json:"from_date"`
	ToDate       time.Time              `json:"to_date"`
	Count        int                    `json:"count"`
	EventResults []*EventResultResponse `json:"event_results"`
}

type CityleagueResultGetByOfficialEventIdResponse struct {
	EventResultResponse
}

type CityleagueResultGetByCityleagueScheduleIdResponse struct {
	CityleagueResultGetResponse
}

type CityleagueResultGetByDateResponse struct {
	CityleagueResultGetResponse
}

type CityleagueResultGetByTermResponse struct {
	CityleagueResultGetResponse
}
