package main

// 実Postgresに対するスモークテスト。
//
// classifyBadgeChange は「DATE列(records.event_date)から読んだ値を TIMESTAMP列
// (user_environment_badges.achieved_at)へ保存して読み戻しても、格納された値としては
// 同じ」という前提に立っている。この前提は列型とドライバの組み合わせで決まるため、
// sqlmockや単体テストでは検証できない。ここが崩れると再実行のたびに全行を上書きし、
// -dry-run が差分の確認として機能しなくなる。
//
// 実行には VSRECORDER_TEST_DATABASE_URL(gormのpostgres DSN)が必要で、未設定の場合は
// スキップされる。`make integration-test` で使い捨てのPostgresを起動して実行できる。

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func TestIntegrationClassifyBadgeChangeAfterRoundTrip(t *testing.T) {
	dsn := os.Getenv("VSRECORDER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VSRECORDER_TEST_DATABASE_URL が未設定のためスキップ(make integration-test で実行できます)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	const (
		userId = "backfillEnvBadgeRoundTripUser"
		// records.id は VARCHAR(26)(ULID)のため26文字に収める。
		recordId = "01JBACKFILLENVBADGEROUND01"
		// environments はdb/schema.sqlで投入済みのものを使う。
		environmentId = "m6"
	)

	// 他のテストのデータを巻き込まないよう、TRUNCATEではなく自分が作った行だけ消す。
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_environment_badges WHERE user_id = ?", userId)
		db.Exec("DELETE FROM records WHERE id = ?", recordId)
	})
	db.Exec("DELETE FROM user_environment_badges WHERE user_id = ?", userId)
	db.Exec("DELETE FROM records WHERE id = ?", recordId)

	require.NoError(t, db.Exec(
		`INSERT INTO records (id, created_at, updated_at, user_id, official_event_id, event_date)
		 VALUES (?, '2026-08-14 10:11:12', '2026-08-14 10:11:12', ?, 1, '2026-08-14')`,
		recordId, userId,
	).Error)

	var record model.Record
	require.NoError(t, db.Where("id = ?", recordId).First(&record).Error)

	ctx := context.Background()
	basisTime := usecase.RecordBasisTime(record.EventDate, record.CreatedAt)
	repo := infrastructure.NewUserEnvironmentBadge(db)
	require.NoError(t, repo.Save(ctx, entity.NewUserEnvironmentBadge(
		userId, environmentId, recordId, "", basisTime, record.CreatedAt,
	)))

	var saved model.UserEnvironmentBadge
	require.NoError(t, db.Where("user_id = ?", userId).First(&saved).Error)

	t.Run("正常系_保存して読み戻した行は上書き対象にならない", func(t *testing.T) {
		require.Equal(t, badgeChangeNone, classifyBadgeChange(&saved, basisTime, record.CreatedAt))
	})

	t.Run("前提_DATE列由来の値は読み戻すとLocationが変わる", func(t *testing.T) {
		// time.Equal(瞬間の一致)で比べていた頃は、この差で全行が上書き対象になっていた。
		// この前提が崩れた(＝Equalでも一致するようになった)場合は sameStoredTime を見直すこと。
		require.False(t, saved.AchievedAt.Equal(basisTime))
		require.Equal(t, basisTime.Format("2006-01-02 15:04:05"), saved.AchievedAt.Format("2006-01-02 15:04:05"))
	})

	t.Run("正常系_達成日時が動く行は上書き対象になる", func(t *testing.T) {
		moved := basisTime.AddDate(0, 0, -1)

		require.Equal(t, badgeChangeUpdate, classifyBadgeChange(&saved, moved, record.CreatedAt))
		require.Equal(t, []string{"achieved_at"}, changedFields(&saved, moved, record.CreatedAt))
	})
}
