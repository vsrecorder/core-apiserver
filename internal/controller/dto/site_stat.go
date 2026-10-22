package dto

type SiteStatResponse struct {
	UserCount     int64 `json:"user_count"`
	MatchCount    int64 `json:"match_count"`
	DeckCodeCount int64 `json:"deck_code_count"`
}
