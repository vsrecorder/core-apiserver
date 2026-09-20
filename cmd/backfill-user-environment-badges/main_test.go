package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

func TestClassifyBadgeChange(t *testing.T) {
	// achieved_at の元になる records.event_date は DATE 列で、UTCラベルの 00:00 として読める。
	achievedAt := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	// TIMESTAMP 列は接続の TimeZone が付いて読み出される。UTCのCIでも差を作れるよう明示する。
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)

	t.Run("正常系_既存行が無い場合はcreateになる", func(t *testing.T) {
		require.Equal(t, badgeChangeCreate, classifyBadgeChange(nil, achievedAt, createdAt))
	})

	t.Run("正常系_既存行と値が同じ場合はnoneになる", func(t *testing.T) {
		existing := &model.UserEnvironmentBadge{AchievedAt: achievedAt, CreatedAt: createdAt}

		require.Equal(t, badgeChangeNone, classifyBadgeChange(existing, achievedAt, createdAt))
	})

	t.Run("正常系_保存して読み戻したことによるLocationの違いはnoneになる", func(t *testing.T) {
		// TIMESTAMP 列は壁時計をそのまま格納するため、UTCラベルで保存した 00:00 は
		// JSTラベルの 00:00 として読み戻る。格納されている値は同じなので上書きしない。
		existing := &model.UserEnvironmentBadge{
			AchievedAt: time.Date(2026, 8, 14, 0, 0, 0, 0, jst),
			CreatedAt:  time.Date(2026, 8, 15, 10, 30, 0, 0, jst),
		}

		require.Equal(t, badgeChangeNone, classifyBadgeChange(existing, achievedAt, createdAt))
	})

	t.Run("正常系_同じ瞬間でも壁時計が違えばupdateになる", func(t *testing.T) {
		// こちらは格納される値が 09:00 と 00:00 で変わるため上書きが要る。
		existing := &model.UserEnvironmentBadge{
			AchievedAt: achievedAt.In(jst),
			CreatedAt:  createdAt.In(jst),
		}

		require.Equal(t, badgeChangeUpdate, classifyBadgeChange(existing, achievedAt, createdAt))
	})

	t.Run("正常系_マイクロ秒未満の差はnoneになる", func(t *testing.T) {
		// PostgreSQLのtimestampはマイクロ秒精度。格納時に落ちる桁で差分と判定しない。
		existing := &model.UserEnvironmentBadge{
			AchievedAt: achievedAt.Add(500 * time.Nanosecond),
			CreatedAt:  createdAt,
		}

		require.Equal(t, badgeChangeNone, classifyBadgeChange(existing, achievedAt, createdAt))
	})

	t.Run("正常系_achieved_atが変わる場合はupdateになる", func(t *testing.T) {
		existing := &model.UserEnvironmentBadge{
			AchievedAt: achievedAt.AddDate(0, 0, 1),
			CreatedAt:  createdAt,
		}

		require.Equal(t, badgeChangeUpdate, classifyBadgeChange(existing, achievedAt, createdAt))
		require.Equal(t, []string{"achieved_at"}, changedFields(existing, achievedAt, createdAt))
	})

	t.Run("正常系_created_atが変わる場合はupdateになる", func(t *testing.T) {
		existing := &model.UserEnvironmentBadge{
			AchievedAt: achievedAt,
			CreatedAt:  createdAt.Add(time.Hour),
		}

		require.Equal(t, badgeChangeUpdate, classifyBadgeChange(existing, achievedAt, createdAt))
		require.Equal(t, []string{"created_at"}, changedFields(existing, achievedAt, createdAt))
	})

	t.Run("正常系_record_idとnotification_idの差は書き込みの理由にならない", func(t *testing.T) {
		// Saveがconflict時に上書きするのはachieved_at/created_atだけなので、
		// これらが違っても書き込む意味が無い(書いても既存値のまま残る)。
		existing := &model.UserEnvironmentBadge{
			RecordId:       "other-record-id",
			NotificationId: "other-notification-id",
			AchievedAt:     achievedAt,
			CreatedAt:      createdAt,
		}

		require.Equal(t, badgeChangeNone, classifyBadgeChange(existing, achievedAt, createdAt))
	})
}

func TestSortBasesForAdoption(t *testing.T) {
	basisTime := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	t.Run("正常系_基準日時の古い順に並ぶ", func(t *testing.T) {
		bases := []matchBasis{
			{matchId: "m2", recordId: "r2", basisTime: basisTime.AddDate(0, 0, 1)},
			{matchId: "m1", recordId: "r1", basisTime: basisTime},
		}

		sortBasesForAdoption(bases)

		require.Equal(t, "m1", bases[0].matchId)
	})

	t.Run("正常系_基準日時が同じなら最も早く作られた対戦が先頭になる", func(t *testing.T) {
		// 同じ記録の中の対戦は基準日時が並ぶ。入力の並び順に関わらず同じ対戦を採用すること。
		newer := matchBasis{matchId: "m2", recordId: "r1", basisTime: basisTime, matchCreatedAt: basisTime.Add(2 * time.Hour)}
		older := matchBasis{matchId: "m1", recordId: "r1", basisTime: basisTime, matchCreatedAt: basisTime.Add(time.Hour)}

		forward := []matchBasis{newer, older}
		backward := []matchBasis{older, newer}
		sortBasesForAdoption(forward)
		sortBasesForAdoption(backward)

		require.Equal(t, "m1", forward[0].matchId)
		require.Equal(t, forward, backward)
	})

	t.Run("正常系_基準日時も作成日時も同じならrecordIdとmatchIdで決まる", func(t *testing.T) {
		// ここまで同じでも順序が入力任せだと、再実行のたびに採用する対戦が変わってしまう。
		a := matchBasis{matchId: "m1", recordId: "r1", basisTime: basisTime, matchCreatedAt: basisTime}
		b := matchBasis{matchId: "m2", recordId: "r1", basisTime: basisTime, matchCreatedAt: basisTime}
		c := matchBasis{matchId: "m0", recordId: "r2", basisTime: basisTime, matchCreatedAt: basisTime}

		bases := []matchBasis{c, b, a}
		sortBasesForAdoption(bases)

		require.Equal(t, []matchBasis{a, b, c}, bases)
	})
}

func TestBackfillStats(t *testing.T) {
	t.Run("正常系_内訳を合算し書き込みが発生する件数を返す", func(t *testing.T) {
		total := backfillStats{}
		total.add(backfillStats{created: 1, updated: 2, unchanged: 3})
		total.add(backfillStats{created: 10, updated: 20, unchanged: 30})

		require.Equal(t, backfillStats{created: 11, updated: 22, unchanged: 33}, total)
		// unchangedは書き込みが発生しないためchangedに含めない。
		require.Equal(t, 33, total.changed())
	})
}
