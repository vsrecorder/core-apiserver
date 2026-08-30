package dto

type OpponentDeckUsageItemResponse struct {
	DeckInfo       string                   `json:"deck_info"`
	Count          int                      `json:"count"`
	UsageRate      float64                  `json:"usage_rate"`
	Wins           int                      `json:"wins"`
	Losses         int                      `json:"losses"`
	WinRate        float64                  `json:"win_rate"`
	PokemonSprites []*PokemonSpriteResponse `json:"pokemon_sprites"`
}

type OpponentDeckUsageStatResponse struct {
	UserId               string                           `json:"user_id"`
	Week                 string                           `json:"week,omitempty"`
	YearMonth            string                           `json:"year_month,omitempty"`
	EnvironmentId        string                           `json:"environment_id,omitempty"`
	Season               string                           `json:"season,omitempty"`
	StandardRegulationId string                           `json:"standard_regulation_id,omitempty"`
	RegulationId         uint                             `json:"regulation_id,omitempty"`
	DeckId               string                           `json:"deck_id,omitempty"`
	TotalMatches         int                              `json:"total_matches"`
	Decks                []*OpponentDeckUsageItemResponse `json:"decks"`
}
