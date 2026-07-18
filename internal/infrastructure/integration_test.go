package infrastructure

// 実Postgresに対するリポジトリ層のスモークテスト。
//
// sqlmockのテストは「GORMが生成するSQL文字列」しか検証できず、db/schema.sql との
// 整合(テーブル名・カラム名・型)は保証されない。ここでは実DBへ読み書きして
// スキーマとの整合を最低限確認する。
//
// 実行には VSRECORDER_TEST_DATABASE_URL(gormのpostgres DSN)が必要で、
// 未設定の場合はスキップされる。`make integration-test` で使い捨てのPostgresを
// 起動してスキーマ適用〜実行〜破棄まで行える。
import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func setupIntegrationDB(t *testing.T, truncateTables ...string) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("VSRECORDER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VSRECORDER_TEST_DATABASE_URL が未設定のためスキップ(make integration-test で実行できます)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	for _, table := range truncateTables {
		require.NoError(t, db.Exec("TRUNCATE TABLE "+table+" CASCADE").Error)
	}

	return db
}

func TestIntegrationUserRepository(t *testing.T) {
	db := setupIntegrationDB(t, "users")
	r := NewUser(db)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	createdAt := time.Now().Local().Truncate(time.Microsecond)

	t.Run("正常系_保存したユーザを取得できる", func(t *testing.T) {
		user := entity.NewUser(uid, createdAt, "テストユーザ", "https://example.com/image.png")

		require.NoError(t, r.Save(context.Background(), user))

		ret, err := r.FindById(context.Background(), uid)

		require.NoError(t, err)
		require.Equal(t, uid, ret.ID)
		require.Equal(t, "テストユーザ", ret.Name)
		require.Equal(t, "https://example.com/image.png", ret.ImageURL)
	})

	t.Run("正常系_削除したユーザはErrRecordNotFoundになる", func(t *testing.T) {
		require.NoError(t, r.Delete(context.Background(), uid))

		_, err := r.FindById(context.Background(), uid)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})
}

func TestIntegrationNotificationRepository(t *testing.T) {
	db := setupIntegrationDB(t, "notifications")
	r := NewNotification(db)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	createdAt := time.Now().Local().Truncate(time.Microsecond)

	t.Run("正常系_保存と取得と既読化が一連で動作する", func(t *testing.T) {
		id1 := "01HD7Y3K8D6FDHMHTZ2GT41TN1"
		id2 := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		n1 := entity.NewNotification(id1, createdAt, uid, "badge", "タイトル1", "本文1", "/badges")
		n2 := entity.NewNotification(id2, createdAt, uid, "designation", "タイトル2", "本文2", "")

		require.NoError(t, r.Save(context.Background(), n1))
		require.NoError(t, r.Save(context.Background(), n2))

		// created_atが同一のときはid降順で安定して返る
		ret, err := r.FindByUserId(context.Background(), uid, 10)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, id2, ret[0].ID)
		require.Equal(t, id1, ret[1].ID)

		count, err := r.CountUnreadByUserId(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, 2, count)

		require.NoError(t, r.MarkAsRead(context.Background(), id1, uid))

		count, err = r.CountUnreadByUserId(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		require.NoError(t, r.MarkAllAsReadByUserId(context.Background(), uid))

		count, err = r.CountUnreadByUserId(context.Background(), uid)
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("異常系_他人の通知の既読化はErrRecordNotFoundになる", func(t *testing.T) {
		err := r.MarkAsRead(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN1", "KBp7roRDZobZg1t0OPzFR1kvLeO2")

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})
}

func TestIntegrationUnofficialEventRepository(t *testing.T) {
	db := setupIntegrationDB(t, "unofficial_events")
	r := NewUnofficialEvent(db)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TU1"

	t.Run("正常系_保存したイベントを取得できる", func(t *testing.T) {
		date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local)
		event := entity.NewUnofficialEvent(id, uid, "自主大会", date)

		require.NoError(t, r.Save(context.Background(), event))

		ret, err := r.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
		require.Equal(t, "自主大会", ret.Title)
		// dateカラム(時刻なし)として保存されるため日付のみ一致を確認する
		require.Equal(t, date.Format(time.DateOnly), ret.Date.Format(time.DateOnly))
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundになる", func(t *testing.T) {
		_, err := r.FindById(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TU9")

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})
}
