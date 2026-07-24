package dto

type WeeklyDeckUsageItemResponse struct {
	Fingerprint    string                   `json:"fingerprint"`
	Count          int                      `json:"count"`
	UsageRate      float64                  `json:"usage_rate"`
	Wins           int                      `json:"wins"`
	Losses         int                      `json:"losses"`
	WinRate        float64                  `json:"win_rate"`
	PokemonSprites []*PokemonSpriteResponse `json:"pokemon_sprites"`
	// Members は「その他」枠に集約された個別変種の内訳。「その他」以外では空のため省略する。
	Members []*WeeklyDeckUsageItemResponse `json:"members,omitempty"`
}

type WeeklyDeckUsageStatResponse struct {
	Week             string                         `json:"week"`
	WeekStart        string                         `json:"week_start"`
	WeekEnd          string                         `json:"week_end"`
	TotalVotes       int                            `json:"total_votes"`
	ContributorCount int                            `json:"contributor_count"`
	Decks            []*WeeklyDeckUsageItemResponse `json:"decks"`
}
