package dto

type WeeklyDeckUsageItemResponse struct {
	Fingerprint     string                   `json:"fingerprint"`
	Label           string                   `json:"label"`
	PrimarySpriteId string                   `json:"primary_sprite_id"`
	Count           int                      `json:"count"`
	UsageRate       float64                  `json:"usage_rate"`
	Wins            int                      `json:"wins"`
	Losses          int                      `json:"losses"`
	WinRate         float64                  `json:"win_rate"`
	PokemonSprites  []*PokemonSpriteResponse `json:"pokemon_sprites"`
}

type WeeklyDeckUsageStatResponse struct {
	Week             string                         `json:"week"`
	WeekStart        string                         `json:"week_start"`
	WeekEnd          string                         `json:"week_end"`
	TotalVotes       int                            `json:"total_votes"`
	ContributorCount int                            `json:"contributor_count"`
	Decks            []*WeeklyDeckUsageItemResponse `json:"decks"`
}
