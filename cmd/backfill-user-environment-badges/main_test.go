package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

func TestClassifyBadgeChange(t *testing.T) {
	achievedAt := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)

	t.Run("正常系_既存行が無い場合はcreateになる", func(t *testing.T) {
		require.Equal(t, badgeChangeCreate, classifyBadgeChange(nil, achievedAt, createdAt))
	})

	t.Run("正常系_既存行と値が同じ場合はnoneになる", func(t *testing.T) {
		existing := &model.UserEnvironmentBadge{AchievedAt: achievedAt, CreatedAt: createdAt}

		require.Equal(t, badgeChangeNone, classifyBadgeChange(existing, achievedAt, createdAt))
	})

	t.Run("正常系_Locationだけが異なる同時刻はnoneになる", func(t *testing.T) {
		// DBドライバの都合でLocationが変わっただけの行を、毎回の再実行で上書きしてしまわないこと。
		// UTCで動くCIでも差を作れるよう、time.Localではなく明示的なオフセットを使う。
		jst := time.FixedZone("Asia/Tokyo", 9*60*60)
		existing := &model.UserEnvironmentBadge{
			AchievedAt: achievedAt.In(jst),
			CreatedAt:  createdAt.In(jst),
		}

		require.Equal(t, badgeChangeNone, classifyBadgeChange(existing, achievedAt, createdAt))
	})

	t.Run("正常系_achieved_atが変わる場合はupdateになる", func(t *testing.T) {
		existing := &model.UserEnvironmentBadge{
			AchievedAt: achievedAt.AddDate(0, 0, 1),
			CreatedAt:  createdAt,
		}

		require.Equal(t, badgeChangeUpdate, classifyBadgeChange(existing, achievedAt, createdAt))
	})

	t.Run("正常系_created_atが変わる場合はupdateになる", func(t *testing.T) {
		existing := &model.UserEnvironmentBadge{
			AchievedAt: achievedAt,
			CreatedAt:  createdAt.Add(time.Hour),
		}

		require.Equal(t, badgeChangeUpdate, classifyBadgeChange(existing, achievedAt, createdAt))
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
