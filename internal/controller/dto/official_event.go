package dto

import "time"

type OfficialEventResponse struct {
	ID              uint      `json:"id"`
	Title           string    `json:"title"`
	Address         string    `json:"address"`
	Venue           string    `json:"venue"`
	Date            string    `json:"date"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at"`
	TypeName        string    `json:"type_name"`
	LeagueTitle     string    `json:"league_title"`
	RegulationTitle string    `json:"regulation_title"`
	CSPFlg          bool      `json:"csp_flg"`
	Capacity        uint      `json:"capacity"`
	ShopId          uint      `json:"shop_id"`
	ShopName        string    `json:"shop_name"`
}

type OfficialEventGetResponse struct {
	TypeId         uint                     `json:"type_id"`
	LeagueType     uint                     `json:"league_type"`
	StartDate      string                   `json:"start_date"`
	EndDate        string                   `json:"end_date"`
	Count          int                      `json:"count"`
	OfficialEvents []*OfficialEventResponse `json:"official_events"`
}

type OfficialEventGetByIdResponse struct {
	OfficialEventResponse
}
