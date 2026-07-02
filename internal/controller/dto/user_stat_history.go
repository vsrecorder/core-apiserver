package dto

type UserStatHistoryItem struct {
	YearMonth    string  `json:"year_month"`
	TotalMatches int     `json:"total_matches"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	WinRate      float64 `json:"win_rate"`
}

type UserStatHistoryResponse struct {
	UserId  string                `json:"user_id"`
	Period  string                `json:"period"`
	Season  string                `json:"season,omitempty"`
	DeckId  string                `json:"deck_id,omitempty"`
	History []UserStatHistoryItem `json:"history"`
}
