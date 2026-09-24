package entity

import "time"

// DeckUsageGrouping は週次デッキ使用率の集計単位（どこまでを「同じデッキ」として束ねるか）。
//
// スプライト（1体目・2体目）の組み合わせが違えば別のデッキ扱いになるのが既定だが、
// 実際の環境では「1体目が同じで2体目だけ違う」変種が多く、同じデッキの派生が
// 別々の行として分散してしまう。1体目だけで束ねる集計も選べるようにしている。
type DeckUsageGrouping string

const (
	// DeckUsageGroupingExact はスプライト（position 1・2）の組み合わせが一致するものだけを
	// 同じデッキとして扱う。既定値。
	DeckUsageGroupingExact DeckUsageGrouping = "exact"

	// DeckUsageGroupingFirstSprite は1体目のスプライトが同じものを同じデッキとして扱う。
	// 2体目が違うだけの派生（同じ軸のデッキ）を1行にまとめて環境の分布を見るための集計単位。
	DeckUsageGroupingFirstSprite DeckUsageGrouping = "first_sprite"
)

// IsValid は集計単位として受け付ける値かどうかを返す。
// 空文字は「未指定＝既定（exact）」として扱うため、ここでは不正とする（呼び出し側で補う）。
func (g DeckUsageGrouping) IsValid() bool {
	switch g {
	case DeckUsageGroupingExact, DeckUsageGroupingFirstSprite:
		return true
	default:
		return false
	}
}

// DeckUsageVariant はプラットフォーム全体の集計における単一のデッキ変種
// （スプライトの集合のみで正規化した指紋。デッキ名等のフリーテキストは使わず、並び順も無視する）を表す。
// DeckUsageGroupingFirstSprite で集計した場合は、指紋が1体目のスプライトだけで決まる。
type DeckUsageVariant struct {
	Fingerprint    string // 正規化済みの集計キー（スプライトIDの集合のみで決まる）
	Count          int
	UsageRate      float64
	Wins           int
	Losses         int
	WinRate        float64
	PokemonSprites []*PokemonSprite
	// Members はこの行に束ねられた内訳。UI ではアコーディオンで展開して一覧表示する。
	//   - 「その他」行: minVariantCount 未満で集約された個別変種
	//   - 1体目でまとめた行(DeckUsageGroupingFirstSprite): 束ねる前の組み合わせ単位の変種
	// 上記以外(組み合わせ一致で集計した通常の行)では nil。
	// 1体目でまとめた集計では「その他」の内訳もまた組み合わせ単位の内訳を持つ(2段)。
	// その他へ落ちた行は1体目しか分からないまま消えるため、何と組んだデッキだったのかを
	// 追えるようにしている。
	// 使用率はどちらも行と同じ全体件数を分母にするため、内訳の合計が行の使用率に一致する。
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
	WeekStart        time.Time         // 集計対象週の開始日（月曜 0時）
	Grouping         DeckUsageGrouping // どの集計単位で束ねた結果か（応答に含めてUIの表示と一致させる）
	TotalVotes       int               // 集計対象となった票の総数（母集団の明示に使う）
	ContributorCount int               // 集計に寄与したユーザー数（母集団の明示に使う）
	Decks            []*DeckUsageVariant
}

func NewWeeklyDeckUsageStat(
	weekStart time.Time,
	grouping DeckUsageGrouping,
	totalVotes int,
	contributorCount int,
	decks []*DeckUsageVariant,
) *WeeklyDeckUsageStat {
	return &WeeklyDeckUsageStat{
		WeekStart:        weekStart,
		Grouping:         grouping,
		TotalVotes:       totalVotes,
		ContributorCount: contributorCount,
		Decks:            decks,
	}
}
