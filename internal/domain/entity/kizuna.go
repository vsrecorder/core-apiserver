package entity

import "math"

/*
 * きずなLv.（0〜255）。デッキとどう歩んできたかを数値化したもの。
 *
 * 設計の要は「勝率を加点しない」こと。強いデッキを持っている人ほど愛情が深い、
 * という設計は破綻しているため。最大の重みを置いた逆境ロイヤルティに至っては、
 * 勝率が低いほど値が上がる。
 *
 * 上限が255なのはポケモン本編の「なつき度」へのオマージュ。
 *
 * 算出アルゴリズムは暫定である。指標の追加・削除、重みの調整、正規化カーブの
 * 見直しが入りうる（vsrecorder/KIZUNA_PLAN.md §4）。同じ対戦記録でも数値が
 * 変わることがあるため、UI には必ずその旨を明記すること。
 */

// KizunaStageKind は「どの格の舞台に連れて行ったか」の分類。
// 分類そのものはデータの事実なので SQL 側で判定し、点数付け（格の重み）は
// 方針なのでこちら側に置く。SQL に点数を書くと、調整のたびにクエリを触ることになる。
type KizunaStageKind int

const (
	// KizunaStageFreeform はイベントに紐づかない対戦（フリー対戦・対戦相手のみ指定）
	KizunaStageFreeform KizunaStageKind = 0
	// 以下は official_events.type_id に対応する。
	// 対応は webapp の RecordCreate.tsx のアイコン出し分けと同じ。
	KizunaStageOfficialLarge      KizunaStageKind = 1 // チャンピオンズリーグ / JCS などの大型大会
	KizunaStageCityLeague         KizunaStageKind = 2
	KizunaStageTrainersLeague     KizunaStageKind = 3
	KizunaStageGymBattle          KizunaStageKind = 4
	KizunaStageOfficialSelfHosted KizunaStageKind = 6 // 公認自主イベント
	KizunaStageClassroom          KizunaStageKind = 7 // ポケモンカード教室などの体験系
	// KizunaStageOfficialUnknown は official_event_id はあるが type_id を引けなかったもの
	KizunaStageOfficialUnknown KizunaStageKind = 90
	KizunaStageTonamel         KizunaStageKind = 91
	KizunaStageUnofficial      KizunaStageKind = 92
)

/*
 * 舞台の格。0.15〜1.0。
 *
 * 「公式イベントかどうか」の二値にしてはいけない。ジムバトルは毎週どこかの店で
 * やっている公式イベントであり、それを最上位に置くと「ジムバトルにしか出ない人」が
 * 満点になる。「ジムバトル100回より、シティリーグの3回のほうが託した」を
 * 成立させるために段を分ける。
 */
var kizunaStageScores = map[KizunaStageKind]float64{
	KizunaStageOfficialLarge:      1.0,
	KizunaStageCityLeague:         0.85,
	KizunaStageTrainersLeague:     0.55,
	KizunaStageOfficialSelfHosted: 0.4,
	KizunaStageGymBattle:          0.3,
	KizunaStageClassroom:          0.2,
	// 種別を引けなかった公式イベントは、最も数の多いジムバトル相当として控えめに見る
	KizunaStageOfficialUnknown: 0.3,
	KizunaStageTonamel:         0.4,
	KizunaStageUnofficial:      0.35,
	KizunaStageFreeform:        0.15,
}

// KizunaStageScore は舞台の格の点数を返す。未知の分類は自由形式と同じ扱いにする。
func KizunaStageScore(kind KizunaStageKind) float64 {
	if score, ok := kizunaStageScores[kind]; ok {
		return score
	}
	return kizunaStageScores[KizunaStageFreeform]
}

const (
	// 託し度の積み上げは、この段（ジムバトル）を超えた分だけを数える。
	// 日常のジムバトルは「託した」とは言わない（回数そのものは同行日数と
	// 逆境ロイヤルティが見ている）。
	kizunaTrustBaselineStage = 0.3
	// 超えた分の合計がこれに達すると積み上げの項は満点（シティリーグ7回ぶん相当）
	kizunaTrustSeriousSaturation = 4.0

	kizunaDaysSaturation        = 30.0  // 同行日数
	kizunaMemoLengthSaturation  = 120.0 // 語り度：メモの平均文字数
	kizunaRebuildSaturation     = 8.0   // 手入れ度：組み直し回数
	kizunaPersistenceSaturation = 40.0  // 逆境ロイヤルティ：継続した対戦数

	// 逆境の判定幅。基準勝率をこれだけ下回ると deficit が最大になる。
	kizunaDeficitRange = 0.2
	// デッキが1つしかない人には比較対象がない。0にするのは酷だが、満点にもしない。
	kizunaSoloDeckDevotion = 0.5

	// きずなLv.の上限（なつき度と同じ）
	KizunaMaxLevel = 255
)

// KizunaMetricKey は指標の識別子。表示名は webapp 側が持つ（UIの文言はUIの責務）。
type KizunaMetricKey string

const (
	KizunaMetricLoyalty   KizunaMetricKey = "loyalty"   // 逆境ロイヤルティ
	KizunaMetricDevotion  KizunaMetricKey = "devotion"  // 一途度
	KizunaMetricCare      KizunaMetricKey = "care"      // 手入れ度
	KizunaMetricDays      KizunaMetricKey = "days"      // 同行日数
	KizunaMetricTrust     KizunaMetricKey = "trust"     // 託し度
	KizunaMetricNarrative KizunaMetricKey = "narrative" // 語り度
)

// 重みは企画書の配分をそのまま使う。合計82%を255点に配り直す。
var kizunaMetricWeights = []struct {
	Key    KizunaMetricKey
	Weight int
}{
	{KizunaMetricLoyalty, 20},
	{KizunaMetricDevotion, 15},
	{KizunaMetricCare, 15},
	{KizunaMetricDays, 12},
	{KizunaMetricTrust, 10},
	{KizunaMetricNarrative, 10},
}

/*
 * KizunaDeckAggregate は、きずなLv.の算出に必要な「デッキ1つぶんの集計済みの素の値」。
 * infrastructure が SQL で集計して詰める。表示用に整形した値は入れない。
 */
type KizunaDeckAggregate struct {
	DeckId string

	// 同行日数：記録のある日付の種類数（棚に置いていた期間は数えない）
	EventDayCount int

	// 託し度：舞台の分類ごとの記録件数
	StageCounts map[KizunaStageKind]int

	// 語り度
	RecordCount     int
	MemoCount       int
	MemoTotalLength int

	// 手入れ度
	DeckCodeCount int // デッキコードの総数（登録時の1件を含む）
	EveCodeCount  int // 大会直前（3日前〜当日昼）に作られたデッキコードの数

	// 逆境ロイヤルティ・一途度
	MatchCount int
	Wins       int
}

// KizunaMetric は指標1つぶんの結果。points の合計がそのままきずなLv.になる。
type KizunaMetric struct {
	Key    KizunaMetricKey
	Weight int
	// Value は 0〜1 に正規化した達成度
	Value float64
	// Points はこの指標で獲得した点、MaxPoints はその満点
	Points    int
	MaxPoints int
}

// KizunaDeck はデッキ1つぶんのきずなLv.
type KizunaDeck struct {
	DeckId  string
	Level   int
	Metrics []*KizunaMetric
}

// Kizuna はユーザーの全デッキぶんのきずなLv.
type Kizuna struct {
	UserId string
	Decks  []*KizunaDeck
}

func NewKizuna(userId string, decks []*KizunaDeck) *Kizuna {
	return &Kizuna{UserId: userId, Decks: decks}
}

func kizunaClamp01(v float64) float64 {
	return math.Min(1, math.Max(0, v))
}

// kizunaLogScale は対数で伸びを鈍らせた 0〜1 の値。saturation に達すると 1 になる。
func kizunaLogScale(value float64, saturation float64) float64 {
	return kizunaClamp01(math.Log(1+math.Max(0, value)) / math.Log(1+saturation))
}

func (a *KizunaDeckAggregate) winRate() float64 {
	if a.MatchCount == 0 {
		return 0
	}
	return float64(a.Wins) / float64(a.MatchCount)
}

/*
 * CalculateKizuna は、ユーザーの全デッキの集計値から各デッキのきずなLv.を求める。
 *
 * 全デッキをまとめて受け取るのは、一途度と逆境ロイヤルティが「ほかのデッキ」を
 * 必要とするため。デッキ単位で呼べる形にすると、指標ごとに他デッキを引き直すことになる。
 */
// kizunaTotals は全デッキの合計。デッキごとの計算で「自分以外」を求めるのに使う。
type kizunaTotals struct {
	usedDeckCount int
	matches       int
	wins          int
}

func CalculateKizuna(aggregates []*KizunaDeckAggregate) []*KizunaDeck {
	// 一途度・逆境ロイヤルティの基準になる、使用されたデッキ全体の状況。
	// 先に合計を1度だけ取っておく。デッキごとに他デッキを走査すると O(デッキ数^2) になる。
	totals := kizunaTotals{}
	for _, a := range aggregates {
		if a.MatchCount > 0 {
			totals.usedDeckCount++
			totals.matches += a.MatchCount
			totals.wins += a.Wins
		}
	}

	decks := make([]*KizunaDeck, 0, len(aggregates))
	for _, a := range aggregates {
		decks = append(decks, calculateKizunaDeck(a, totals))
	}

	return decks
}

func calculateKizunaDeck(
	a *KizunaDeckAggregate,
	totals kizunaTotals,
) *KizunaDeck {
	// ── 同行日数：実際に会場へ連れて行った日数
	daysValue := kizunaLogScale(float64(a.EventDayCount), kizunaDaysSaturation)

	// ── 託し度：どの格の舞台に連れて行ったか
	//
	// 「いちばん大きな舞台に持ち込んだか」（最高到達点）を主に、
	// 「大きな舞台に何度立ったか」（積み上げ）を従に見る。
	//
	// 従を「平均」にしてはいけない。平均は記録が増えるほど下がるため、
	// ジムバトルの記録を足すたびに託し度が下がる、シティ1回だけの人が
	// シティ3回＋ジム27回の人を上回る、といった倒錯が起きる。
	topStageScore := 0.0
	seriousSum := 0.0
	for kind, count := range a.StageCounts {
		if count <= 0 {
			continue
		}
		score := KizunaStageScore(kind)
		if score > topStageScore {
			topStageScore = score
		}
		seriousSum += math.Max(0, score-kizunaTrustBaselineStage) * float64(count)
	}
	trustValue := kizunaClamp01(
		0.6*topStageScore + 0.4*kizunaLogScale(seriousSum, kizunaTrustSeriousSaturation),
	)

	// ── 語り度：メモを書き残したか（記入率 × 熱量）
	narrativeValue := 0.0
	if a.RecordCount > 0 && a.MemoCount > 0 {
		memoRate := float64(a.MemoCount) / float64(a.RecordCount)
		memoAvgLength := float64(a.MemoTotalLength) / float64(a.MemoCount)
		narrativeValue = memoRate * kizunaLogScale(memoAvgLength, kizunaMemoLengthSaturation)
	}

	// ── 手入れ度：組み直した回数と、その時刻（大会前の調整を重く見る）
	//
	// デッキコードは登録時に必ず1件できる。それを「組み直し」に数えると、
	// 一度も触っていないデッキにも点が入ってしまうため1を引く。
	rebuildCount := a.DeckCodeCount - 1
	if rebuildCount < 0 {
		rebuildCount = 0
	}
	eveRate := 0.0
	if a.DeckCodeCount > 0 {
		eveRate = float64(a.EveCodeCount) / float64(a.DeckCodeCount)
	}
	careValue := kizunaClamp01(
		kizunaLogScale(float64(rebuildCount), kizunaRebuildSaturation) * (0.6 + 0.4*eveRate),
	)

	// ── 一途度：他に選べたのに、それでも選んだか
	//
	// 見るべきは「均等に使ったらこうなるはず」（= 1 / 使ったデッキ数）からの寄せ方。
	// 同じ占有率でも、選択肢が多いほど価値が上がる
	// （1個持ちの100%より、10個持ちの70%のほうが深い）。
	devotionValue := 0.0
	share := 0.0
	if a.MatchCount > 0 && totals.matches > 0 {
		share = float64(a.MatchCount) / float64(totals.matches)
	}
	switch {
	case a.MatchCount == 0:
		devotionValue = 0
	case totals.usedDeckCount <= 1:
		devotionValue = kizunaSoloDeckDevotion
	default:
		evenShare := 1 / float64(totals.usedDeckCount)
		devotionValue = kizunaClamp01((share - evenShare) / (1 - evenShare))
	}

	// ── 逆境ロイヤルティ：勝てなくても使い続けたか
	// この指標だけは勝率が低いほど上がる。
	persistence := kizunaLogScale(float64(a.MatchCount), kizunaPersistenceSaturation)

	// 逆境の基準は「ほかのデッキでの勝率」。本人の全体勝率を基準にすると、
	// そのデッキを主に使っている人ほど基準が自分自身に引きずられ、
	// 1デッキしか使っていない人は勝率が何%でも deficit が 0 になってしまう
	// （いちばん一途な人ほど逆境を検知できない、という本末転倒になる）。
	//
	// 「自分以外」は合計から自分を引いて求める。デッキIDは一意なので、
	// 他デッキを走査して足し合わせるのと結果は同じで、走査のぶんだけ速い。
	otherMatches := totals.matches - a.MatchCount
	otherWins := totals.wins - a.Wins
	// ほかに使ったデッキがなければ、五分（50%）を基準にする
	baselineWinRate := 0.5
	if otherMatches > 0 {
		baselineWinRate = float64(otherWins) / float64(otherMatches)
	}
	deficit := kizunaClamp01((baselineWinRate - a.winRate()) / kizunaDeficitRange)
	loyaltyValue := persistence * (0.5 + 0.5*deficit)

	values := map[KizunaMetricKey]float64{
		KizunaMetricLoyalty:   loyaltyValue,
		KizunaMetricDevotion:  devotionValue,
		KizunaMetricCare:      careValue,
		KizunaMetricDays:      daysValue,
		KizunaMetricTrust:     trustValue,
		KizunaMetricNarrative: narrativeValue,
	}

	/*
	 * 重みを255点に配り、指標ごとの満点にする（20:15:15:12:10:10 → 62:47:47:37:31:31）。
	 * 獲得点は満点 × 達成度で、きずなLv.はその合計そのもの。
	 * 「重み付き和を最後に255倍する」と結果は同じだが、内訳の足し算が合わなくなる
	 * （表示のために丸めるため）。合計＝きずなLv.であることを式の側で保証する。
	 */
	totalWeight := 0
	for _, w := range kizunaMetricWeights {
		totalWeight += w.Weight
	}

	metrics := make([]*KizunaMetric, 0, len(kizunaMetricWeights))
	level := 0
	for _, w := range kizunaMetricWeights {
		maxPoints := int(math.Round(float64(w.Weight) / float64(totalWeight) * KizunaMaxLevel))
		points := int(math.Round(values[w.Key] * float64(maxPoints)))
		level += points
		metrics = append(metrics, &KizunaMetric{
			Key:       w.Key,
			Weight:    w.Weight,
			Value:     values[w.Key],
			Points:    points,
			MaxPoints: maxPoints,
		})
	}

	return &KizunaDeck{DeckId: a.DeckId, Level: level, Metrics: metrics}
}
