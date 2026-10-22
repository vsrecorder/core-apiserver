package entity

// SiteStat はサービス全体の規模を表す件数(トップページの統計カウンタに出す値)。
// いずれも未削除の行だけを数えた現在値。
type SiteStat struct {
	UserCount     int64
	MatchCount    int64
	DeckCodeCount int64
}

func NewSiteStat(
	userCount int64,
	matchCount int64,
	deckCodeCount int64,
) *SiteStat {
	return &SiteStat{
		UserCount:     userCount,
		MatchCount:    matchCount,
		DeckCodeCount: deckCodeCount,
	}
}
