package dto

type RecentMatchItem struct {
	Sequence          int                      `json:"sequence"`
	EventDate         string                   `json:"event_date"`
	DeckId            string                   `json:"deck_id"`
	OpponentsDeckInfo string                   `json:"opponents_deck_info"`
	Victory           bool                     `json:"victory"`
	RollingWinRate    float64                  `json:"rolling_win_rate"`
	EnvironmentId     string                   `json:"environment_id,omitempty"`
	EnvironmentTitle  string                   `json:"environment_title,omitempty"`
	PokemonSprites    []*PokemonSpriteResponse `json:"pokemon_sprites"`
}

type RecentMatchStatResponse struct {
	UserId       string            `json:"user_id"`
	Count        int               `json:"count"`
	DeckId       string            `json:"deck_id,omitempty"`
	TotalMatches int               `json:"total_matches"`
	Wins         int               `json:"wins"`
	WinRate      float64           `json:"win_rate"`
	Matches      []RecentMatchItem `json:"matches"`
}
