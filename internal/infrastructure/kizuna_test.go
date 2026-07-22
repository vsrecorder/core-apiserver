package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

/*
 * 舞台の分類は SQL 側で判定し、点数付けは entity 側が持つ、という分担にしている。
 * そのぶん「SQL に書いた数値」と「entity の定数」がずれると、ジムバトルの記録が
 * 大型大会として数えられる、といった静かな事故になる。ここで固定する。
 */
func TestKizunaStageKindSQLLiterals(t *testing.T) {
	// official_events.type_id をそのまま分類に使うもの。
	// SQL の IN 句に並べた値と一致していること。
	inClause := []entity.KizunaStageKind{
		entity.KizunaStageOfficialLarge,
		entity.KizunaStageCityLeague,
		entity.KizunaStageTrainersLeague,
		entity.KizunaStageGymBattle,
		entity.KizunaStageOfficialSelfHosted,
		entity.KizunaStageClassroom,
	}

	want := make([]string, 0, len(inClause))
	for _, kind := range inClause {
		want = append(want, fmt.Sprintf("%d", int(kind)))
	}
	require.Contains(t, kizunaStageKindExpr, "IN ("+strings.Join(want, ",")+")",
		"SQL の IN 句と entity.KizunaStageKind がずれている")

	// 個別のセンチネル値
	for _, tt := range []struct {
		name string
		kind entity.KizunaStageKind
	}{
		{"種別を引けなかった公式イベント", entity.KizunaStageOfficialUnknown},
		{"Tonamel", entity.KizunaStageTonamel},
		{"自主大会", entity.KizunaStageUnofficial},
	} {
		require.Contains(t, kizunaStageKindExpr, fmt.Sprintf("%d", int(tt.kind)),
			"%s の分類値が SQL に無い", tt.name)
	}

	// イベントに紐づかない対戦は 0
	require.Contains(t, kizunaStageKindExpr, "ELSE 0")
	require.Equal(t, entity.KizunaStageKind(0), entity.KizunaStageFreeform)
}

// ── 実Postgresでの検証 ───────────────────────────────────────
// GORM が組み立てた SQL が db/schema.sql に対して実際に動くかは、sqlmock では
// 分からない（sqlmock は生成された文字列を照合するだけ）。集計の中身まで確かめる。

func insertKizunaFixtures(t *testing.T, db *gorm.DB, userId string) {
	t.Helper()

	// 公式イベント：シティリーグ(type_id=2)とジムバトル(type_id=4)
	require.NoError(t, db.Exec(
		`INSERT INTO official_events (id, title, address, date, type_id) VALUES
		 (2001, 'シティリーグ', '東京', '2026-03-01', 2),
		 (2002, 'ジムバトル', '東京', '2026-03-02', 4)`).Error)

	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.Local)

	// デッキ2つ。d2 は記録が無い（登録しただけ）
	require.NoError(t, db.Exec(
		`INSERT INTO decks (id, created_at, updated_at, user_id, name) VALUES
		 ('deck-01', ?, ?, ?, '主力'),
		 ('deck-02', ?, ?, ?, '未使用')`,
		now, now, userId, now, now, userId).Error)

	// 記録：同じ日に2件（同行日数は日付の種類で数えるので1日と数える）＋別日に2件
	require.NoError(t, db.Exec(
		`INSERT INTO records (id, created_at, updated_at, user_id, deck_id, official_event_id, event_date, memo, ignore_stats_flg) VALUES
		 ('rec-01', ?, ?, ?, 'deck-01', 2001, '2026-03-01', 'よく回った', false),
		 ('rec-02', ?, ?, ?, 'deck-01', 2002, '2026-03-01', '', false),
		 ('rec-03', ?, ?, ?, 'deck-01', 2002, '2026-03-05', 'あああ', false),
		 -- 集計対象外の記録も数える（きずなは勝率を見ないため、除外する理由がない）
		 ('rec-04', ?, ?, ?, 'deck-01', 2002, '2026-03-09', 'これも数える', true)`,
		now, now, userId, now, now, userId, now, now, userId, now, now, userId).Error)

	// デッキコード：3件。うち1件はシティリーグ(3/1)の前日＝大会に向けた調整とみなす
	require.NoError(t, db.Exec(
		`INSERT INTO deck_codes (id, created_at, updated_at, user_id, deck_id, code) VALUES
		 ('code-01', ?, ?, ?, 'deck-01', 'aaaaa'),
		 ('code-02', ?, ?, ?, 'deck-01', 'bbbbb'),
		 ('code-03', ?, ?, ?, 'deck-01', 'ccccc')`,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local), now, userId,
		time.Date(2020, 1, 2, 0, 0, 0, 0, time.Local), now, userId,
		time.Date(2026, 2, 28, 22, 0, 0, 0, time.Local), now, userId).Error)

	// 対戦：4戦1勝。match-01 は BO3 で games が2行付く（match 単位で数えられること）
	// match-04 は集計対象外の記録(rec-04)に紐づく対戦で、これも数える
	require.NoError(t, db.Exec(
		`INSERT INTO matches (id, created_at, updated_at, user_id, record_id, deck_id,
		                      bo3_flg, qualifying_round_flg, final_tournament_flg, victory_flg) VALUES
		 ('match-01', ?, ?, ?, 'rec-01', 'deck-01', true,  true, false, true),
		 ('match-02', ?, ?, ?, 'rec-01', 'deck-01', false, true, false, false),
		 ('match-03', ?, ?, ?, 'rec-03', 'deck-01', false, true, false, false),
		 ('match-04', ?, ?, ?, 'rec-04', 'deck-01', false, true, false, false)`,
		now, now, userId, now, now, userId, now, now, userId, now, now, userId).Error)

	require.NoError(t, db.Exec(
		`INSERT INTO games (id, created_at, updated_at, user_id, match_id, go_first, winning_flg) VALUES
		 ('game-01', ?, ?, ?, 'match-01', true,  true),
		 ('game-02', ?, ?, ?, 'match-01', false, true)`,
		now, now, userId, now, now, userId).Error)
}

func TestIntegrationKizunaRepository(t *testing.T) {
	db := setupIntegrationDB(t,
		"games", "matches", "records", "deck_codes", "decks", "official_events")
	r := NewKizuna(db)

	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	insertKizunaFixtures(t, db, userId)

	aggregates, err := r.FindKizunaDeckAggregates(context.Background(), userId)
	require.NoError(t, err)

	byDeck := map[string]*entity.KizunaDeckAggregate{}
	for _, a := range aggregates {
		byDeck[a.DeckId] = a
	}

	t.Run("正常系_記録のあるデッキの集計が取れる", func(t *testing.T) {
		a := byDeck["deck-01"]
		require.NotNil(t, a)

		// 3/1 に2件・3/5 に1件・3/9 に1件。集計対象外(rec-04)も数える
		require.Equal(t, 3, a.EventDayCount)
		require.Equal(t, 4, a.RecordCount)
		// メモがあるのは rec-01・rec-03 と、集計対象外の rec-04
		require.Equal(t, 3, a.MemoCount)
		require.Equal(t,
			len([]rune("よく回った"))+len([]rune("あああ"))+len([]rune("これも数える")),
			a.MemoTotalLength)

		// シティリーグ1件・ジムバトル3件（rec-02, rec-03 と集計対象外の rec-04）
		require.Equal(t, 1, a.StageCounts[entity.KizunaStageCityLeague])
		require.Equal(t, 3, a.StageCounts[entity.KizunaStageGymBattle])

		require.Equal(t, 3, a.DeckCodeCount)
		// 3/1 のシティリーグの前日に作った code-03 だけが該当する
		require.Equal(t, 1, a.EveCodeCount)

		// 集計対象外の記録(rec-04)に紐づく match-04 も含めて4戦1勝
		require.Equal(t, 4, a.MatchCount)
		require.Equal(t, 1, a.Wins)
	})

	t.Run("正常系_記録が無いデッキもゼロ値で返る", func(t *testing.T) {
		a := byDeck["deck-02"]
		require.NotNil(t, a, "記録が無いデッキも「出会ったばかり」として返す必要がある")

		require.Equal(t, 0, a.RecordCount)
		require.Equal(t, 0, a.MatchCount)
		require.Equal(t, 0, a.DeckCodeCount)
	})

	t.Run("正常系_他人のデッキは含まれない", func(t *testing.T) {
		other, err := r.FindKizunaDeckAggregates(context.Background(), "other-user")
		require.NoError(t, err)
		require.Empty(t, other)
	})
}
