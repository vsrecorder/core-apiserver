package dto

type UserStatResponse struct {
	UserId        string  `json:"user_id"`
	YearMonth     string  `json:"year_month,omitempty"`
	EnvironmentId string  `json:"environment_id,omitempty"`
	TotalMatches  int     `json:"total_matches"`
	Wins          int     `json:"wins"`
	Losses        int     `json:"losses"`
	WinRate       float64 `json:"win_rate"`
}
