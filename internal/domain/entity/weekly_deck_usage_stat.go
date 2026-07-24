package entity

import "time"

// DeckUsageVariant はプラットフォーム全体の集計における単一のデッキ変種
// （スプライトの集合のみで正規化した指紋。デッキ名等のフリーテキストは使わず、並び順も無視する）を表す。
type DeckUsageVariant struct {
	Fingerprint    string // 正規化済みの集計キー（スプライトIDの集合のみで決まる）
	Count          int
	UsageRate      float64
	Wins           int
	Losses         int
	WinRate        float64
	PokemonSprites []*PokemonSprite
	// Members は「その他」枠に集約された個別変種の内訳。
	// 「その他」以外の変種では nil。UI ではアコーディオンで展開し、少数変種も
	// 個別に一覧表示するために使う。
	Members []*DeckUsageVariant
	// PreviousRank は前週の同じ指紋の順位(個別表示された変種のみ・1始まり)。
	// 前週に個別表示されていない(圏外・「その他」集約・新登場)場合は nil。
	// 「その他」行は順位を持たないため常に nil。
	PreviousRank *int
	// PreviousUsageRate / PreviousWinRate は前週の同じ指紋の使用率・勝率。
	// 前週に個別表示されていなければ nil。「その他」行は前週の「その他」と比較する。
	PreviousUsageRate *float64
	PreviousWinRate   *float64
}

func NewDeckUsageVariant(
	fingerprint string,
	count int,
	usageRate float64,
	wins int,
	losses int,
	winRate float64,
	pokemonSprites []*PokemonSprite,
) *DeckUsageVariant {
	return &DeckUsageVariant{
		Fingerprint:    fingerprint,
		Count:          count,
		UsageRate:      usageRate,
		Wins:           wins,
		Losses:         losses,
		WinRate:        winRate,
		PokemonSprites: pokemonSprites,
	}
}

// WeeklyDeckUsageStat はある週のプラットフォーム全体のデッキ使用率集計結果を表す。
type WeeklyDeckUsageStat struct {
	WeekStart        time.Time // 集計対象週の開始日（月曜 0時）
	TotalVotes       int       // 集計対象となった票の総数（母集団の明示に使う）
	ContributorCount int       // 集計に寄与したユーザー数（母集団の明示に使う）
	Decks            []*DeckUsageVariant
}

func NewWeeklyDeckUsageStat(
	weekStart time.Time,
	totalVotes int,
	contributorCount int,
	decks []*DeckUsageVariant,
) *WeeklyDeckUsageStat {
	return &WeeklyDeckUsageStat{
		WeekStart:        weekStart,
		TotalVotes:       totalVotes,
		ContributorCount: contributorCount,
		Decks:            decks,
	}
}
