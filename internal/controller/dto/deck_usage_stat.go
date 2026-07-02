package dto

type DeckUsageItemResponse struct {
	DeckId         string                   `json:"deck_id"`
	Name           string                   `json:"name"`
	Count          int                      `json:"count"`
	UsageRate      float64                  `json:"usage_rate"`
	Wins           int                      `json:"wins"`
	Losses         int                      `json:"losses"`
	WinRate        float64                  `json:"win_rate"`
	PokemonSprites []*PokemonSpriteResponse `json:"pokemon_sprites"`
}

type DeckUsageStatResponse struct {
	UserId        string                   `json:"user_id"`
	YearMonth     string                   `json:"year_month,omitempty"`
	EnvironmentId string                   `json:"environment_id,omitempty"`
	Season        string                   `json:"season,omitempty"`
	RegulationId  string                   `json:"regulation_id,omitempty"`
	TotalRecords  int                      `json:"total_records"`
	Decks         []*DeckUsageItemResponse `json:"decks"`
}
