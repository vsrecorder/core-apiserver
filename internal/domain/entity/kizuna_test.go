package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

/*
 * きずなLv.の算出は webapp/src/app/utils/kizuna.ts のリファレンス実装からの移植である。
 * 期待値は、同じ入力をリファレンス実装に通して得た出力をそのまま置いている。
 * 式を変更するときは、必ず両方を同時に更新して値が一致することを確かめること
 * （webapp 側は /kizuna の試算シミュレーターが使っている）。
 */

func TestCalculateKizuna(t *testing.T) {
	tests := map[string]struct {
		aggregates []*KizunaDeckAggregate
		// deckId → 期待するきずなLv.
		wantLevels map[string]int
		// deckId → 指標ごとの獲得点
		wantPoints map[string]map[KizunaMetricKey]int
	}{
		"ジムバトルだけの初心者": {
			aggregates: []*KizunaDeckAggregate{
				{
					DeckId:        "d1",
					EventDayCount: 6,
					StageCounts:   map[KizunaStageKind]int{KizunaStageGymBattle: 6},
					RecordCount:   6,
					DeckCodeCount: 1,
					MatchCount:    6,
					Wins:          3,
				},
			},
			wantLevels: map[string]int{"d1": 67},
			wantPoints: map[string]map[KizunaMetricKey]int{
				"d1": {
					KizunaMetricLoyalty:   16,
					KizunaMetricDevotion:  24,
					KizunaMetricCare:      0,
					KizunaMetricDays:      21,
					KizunaMetricTrust:     6,
					KizunaMetricNarrative: 0,
				},
			},
		},
		"勝てないのに握り続けた1デッキ": {
			aggregates: []*KizunaDeckAggregate{
				{
					DeckId:        "d1",
					EventDayCount: 18,
					StageCounts: map[KizunaStageKind]int{
						KizunaStageGymBattle:  20,
						KizunaStageCityLeague: 4,
					},
					RecordCount:     24,
					MemoCount:       14,
					MemoTotalLength: 14 * 40,
					DeckCodeCount:   13,
					EveCodeCount:    4,
					MatchCount:      24,
					Wins:            8,
				},
			},
			wantLevels: map[string]int{"d1": 178},
			wantPoints: map[string]map[KizunaMetricKey]int{
				"d1": {
					KizunaMetricLoyalty:   49,
					KizunaMetricDevotion:  24,
					KizunaMetricCare:      34,
					KizunaMetricDays:      32,
					KizunaMetricTrust:     25,
					KizunaMetricNarrative: 14,
				},
			},
		},
		"主力とサブの2デッキ": {
			aggregates: []*KizunaDeckAggregate{
				{
					DeckId:        "d1",
					EventDayCount: 18,
					StageCounts: map[KizunaStageKind]int{
						KizunaStageGymBattle:  20,
						KizunaStageCityLeague: 4,
					},
					RecordCount:     24,
					MemoCount:       14,
					MemoTotalLength: 14 * 40,
					DeckCodeCount:   13,
					EveCodeCount:    4,
					MatchCount:      24,
					Wins:            8,
				},
				{
					DeckId:        "d2",
					EventDayCount: 5,
					StageCounts:   map[KizunaStageKind]int{KizunaStageGymBattle: 5},
					RecordCount:   5,
					DeckCodeCount: 1,
					MatchCount:    10,
					Wins:          7,
				},
			},
			// サブを持つと一途度が下がる（24→19）一方、
			// 勝率の高いサブが基準になるため逆境ロイヤルティは上がる（49→54）
			wantLevels: map[string]int{"d1": 178, "d2": 45},
			wantPoints: map[string]map[KizunaMetricKey]int{
				"d1": {
					KizunaMetricLoyalty:   54,
					KizunaMetricDevotion:  19,
					KizunaMetricCare:      34,
					KizunaMetricDays:      32,
					KizunaMetricTrust:     25,
					KizunaMetricNarrative: 14,
				},
				"d2": {
					KizunaMetricLoyalty:   20,
					KizunaMetricDevotion:  0,
					KizunaMetricCare:      0,
					KizunaMetricDays:      19,
					KizunaMetricTrust:     6,
					KizunaMetricNarrative: 0,
				},
			},
		},
		"登録しただけで記録が無いデッキ": {
			aggregates: []*KizunaDeckAggregate{
				{DeckId: "d1", DeckCodeCount: 1},
			},
			wantLevels: map[string]int{"d1": 0},
			wantPoints: map[string]map[KizunaMetricKey]int{
				"d1": {
					KizunaMetricLoyalty:   0,
					KizunaMetricDevotion:  0,
					KizunaMetricCare:      0,
					KizunaMetricDays:      0,
					KizunaMetricTrust:     0,
					KizunaMetricNarrative: 0,
				},
			},
		},
		"大型大会まで連れて行った主力": {
			aggregates: []*KizunaDeckAggregate{
				{
					DeckId:        "d1",
					EventDayCount: 30,
					StageCounts: map[KizunaStageKind]int{
						KizunaStageOfficialLarge: 2,
						KizunaStageCityLeague:    6,
						KizunaStageGymBattle:     30,
					},
					RecordCount:     38,
					MemoCount:       38,
					MemoTotalLength: 38 * 150,
					DeckCodeCount:   20,
					EveCodeCount:    15,
					MatchCount:      60,
					Wins:            20,
				},
				{
					DeckId:        "d2",
					EventDayCount: 8,
					StageCounts:   map[KizunaStageKind]int{KizunaStageGymBattle: 8},
					RecordCount:   8,
					DeckCodeCount: 2,
					MatchCount:    20,
					Wins:          14,
				},
			},
			wantLevels: map[string]int{"d1": 227, "d2": 64},
			wantPoints: map[string]map[KizunaMetricKey]int{
				"d1": {
					KizunaMetricLoyalty:   62,
					KizunaMetricDevotion:  24,
					KizunaMetricCare:      42,
					KizunaMetricDays:      37,
					KizunaMetricTrust:     31,
					KizunaMetricNarrative: 31,
				},
				"d2": {
					KizunaMetricLoyalty:   25,
					KizunaMetricDevotion:  0,
					KizunaMetricCare:      9,
					KizunaMetricDays:      24,
					KizunaMetricTrust:     6,
					KizunaMetricNarrative: 0,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			decks := CalculateKizuna(tt.aggregates)
			require.Len(t, decks, len(tt.aggregates))

			for _, deck := range decks {
				require.Equal(t, tt.wantLevels[deck.DeckId], deck.Level, "deckId=%s のきずなLv.", deck.DeckId)

				sum := 0
				for _, m := range deck.Metrics {
					require.Equal(t, tt.wantPoints[deck.DeckId][m.Key], m.Points, "deckId=%s の指標=%s", deck.DeckId, m.Key)
					sum += m.Points
				}
				// 内訳の合計が、そのままきずなLv.になっていること
				require.Equal(t, deck.Level, sum, "deckId=%s の内訳合計", deck.DeckId)
			}
		})
	}
}

// 満点の配分が255ちょうどになること。ここが崩れると最高段階「最高の相棒」に到達できなくなる。
func TestCalculateKizunaMaxPointsSumsTo255(t *testing.T) {
	decks := CalculateKizuna([]*KizunaDeckAggregate{{DeckId: "d1"}})
	require.Len(t, decks, 1)

	sum := 0
	for _, m := range decks[0].Metrics {
		sum += m.MaxPoints
	}
	require.Equal(t, KizunaMaxLevel, sum)
}

// 勝率はきずなLv.に加点されない。それどころか、負けているほど逆境ロイヤルティは上がる。
func TestCalculateKizunaRewardsLosing(t *testing.T) {
	base := func(wins int) []*KizunaDeckAggregate {
		return []*KizunaDeckAggregate{
			{
				DeckId:        "d1",
				EventDayCount: 20,
				StageCounts:   map[KizunaStageKind]int{KizunaStageGymBattle: 40},
				RecordCount:   40,
				DeckCodeCount: 5,
				MatchCount:    40,
				Wins:          wins,
			},
			// 基準になる「ほかのデッキの勝率」を五分に固定する
			{
				DeckId:        "d2",
				EventDayCount: 5,
				StageCounts:   map[KizunaStageKind]int{KizunaStageGymBattle: 10},
				RecordCount:   10,
				DeckCodeCount: 1,
				MatchCount:    10,
				Wins:          5,
			},
		}
	}

	loser := CalculateKizuna(base(10))  // 勝率25%
	winner := CalculateKizuna(base(30)) // 勝率75%

	require.Greater(t, loser[0].Level, winner[0].Level, "勝てないほうがきずなLv.は高くなる")
}
