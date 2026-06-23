package dto

type UserStatResponse struct {
	UserId             string  `json:"user_id"`
	YearMonth          string  `json:"year_month,omitempty"`
	EnvironmentId      string  `json:"environment_id,omitempty"`
	TotalRecords       int     `json:"total_records"`
	OfficialEventCount int     `json:"official_event_count"`
	TonamelEventCount  int     `json:"tonamel_event_count"`
	TotalMatches       int     `json:"total_matches"`
	Wins               int     `json:"wins"`
	Losses             int     `json:"losses"`
	WinRate            float64 `json:"win_rate"`
}
