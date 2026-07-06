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
