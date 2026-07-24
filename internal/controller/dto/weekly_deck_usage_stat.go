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
	// 前週の同じ指紋の順位・使用率・勝率(UI の上昇/下降表示用)。
	// 前週に個別表示されていない(圏外・「その他」集約・新登場)場合は省略される。
	PreviousRank      *int     `json:"previous_rank,omitempty"`
	PreviousUsageRate *float64 `json:"previous_usage_rate,omitempty"`
	PreviousWinRate   *float64 `json:"previous_win_rate,omitempty"`
}

type WeeklyDeckUsageStatResponse struct {
	Week             string                         `json:"week"`
	WeekStart        string                         `json:"week_start"`
	WeekEnd          string                         `json:"week_end"`
	TotalVotes       int                            `json:"total_votes"`
	ContributorCount int                            `json:"contributor_count"`
	Decks            []*WeeklyDeckUsageItemResponse `json:"decks"`
}
