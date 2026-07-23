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
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
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

// 退会時の一括削除(DeleteByUserId)が、退会者の関連データを漏れなく消し、かつ
// 他ユーザのデータを巻き込まないことを実DBで確認する。
// 一括削除はサブクエリで対象を絞るため、条件を1つ間違えると他人のデータまで
// 消えうる。sqlmockではSQL文字列しか見られないので、ここで実際の結果を検証する。
func TestIntegrationDeleteByUserId(t *testing.T) {
	db := setupIntegrationDB(t, "games", "matches", "records", "deck_codes", "decks", "unofficial_events")

	const (
		withdrawUid = "zor5SLfEfwfZ90yRVXzlxBEFARy2" // 退会するユーザ
		otherUid    = "CeQ0Oa9g9uRThL11lj4l45VAg8p1" // 無関係なユーザ
	)

	now := time.Now().Local().Truncate(time.Microsecond)

	// --- 退会者のデータ
	// デッキ2つ(うち1つはアーカイブ済み)と、それぞれに紐づくデッキコード
	require.NoError(t, db.Create(&model.Deck{ID: "deck-w1", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, Name: "デッキ1"}).Error)
	require.NoError(t, db.Create(&model.Deck{ID: "deck-w2", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, Name: "デッキ2", ArchivedAt: sql.NullTime{Time: now, Valid: true}}).Error)
	require.NoError(t, db.Create(&model.DeckCode{ID: "dc-w1", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, DeckId: "deck-w1", Code: "aaaa"}).Error)
	require.NoError(t, db.Create(&model.DeckCode{ID: "dc-w2", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, DeckId: "deck-w2", Code: "bbbb"}).Error)

	// 自由形式イベントを参照する記録と、通常の記録
	require.NoError(t, db.Create(&model.UnofficialEvent{ID: "ue-w1", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, Title: "自主大会", Date: now}).Error)
	require.NoError(t, db.Create(&model.Record{ID: "rec-w1", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, DeckId: "deck-w1", EventDate: now, UnofficialEventId: "ue-w1"}).Error)
	require.NoError(t, db.Create(&model.Record{ID: "rec-w2", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, DeckId: "deck-w1", EventDate: now}).Error)

	require.NoError(t, db.Create(&model.Match{ID: "mat-w1", CreatedAt: now, UpdatedAt: now, RecordId: "rec-w1", UserId: withdrawUid}).Error)
	require.NoError(t, db.Create(&model.Match{ID: "mat-w2", CreatedAt: now, UpdatedAt: now, RecordId: "rec-w2", UserId: withdrawUid}).Error)
	require.NoError(t, db.Create(&model.Game{ID: "gam-w1", CreatedAt: now, UpdatedAt: now, MatchId: "mat-w1", UserId: withdrawUid}).Error)
	require.NoError(t, db.Create(&model.Game{ID: "gam-w2", CreatedAt: now, UpdatedAt: now, MatchId: "mat-w2", UserId: withdrawUid}).Error)

	// 他人のデッキに対して退会者が作ったデッキコード(deck経由では消えない)
	require.NoError(t, db.Create(&model.Deck{ID: "deck-o1", CreatedAt: now, UpdatedAt: now, UserId: otherUid, Name: "他人のデッキ"}).Error)
	require.NoError(t, db.Create(&model.DeckCode{ID: "dc-w3", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, DeckId: "deck-o1", Code: "cccc"}).Error)

	// 退会者のデッキに対して他人が作ったデッキコード(デッキが消える以上、残さない)
	require.NoError(t, db.Create(&model.DeckCode{ID: "dc-o1", CreatedAt: now, UpdatedAt: now, UserId: otherUid, DeckId: "deck-w1", Code: "dddd"}).Error)

	// --- 巻き込まれてはいけない他人のデータ
	require.NoError(t, db.Create(&model.UnofficialEvent{ID: "ue-o1", CreatedAt: now, UpdatedAt: now, UserId: otherUid, Title: "他人の自主大会", Date: now}).Error)
	require.NoError(t, db.Create(&model.Record{ID: "rec-o1", CreatedAt: now, UpdatedAt: now, UserId: otherUid, DeckId: "deck-o1", EventDate: now, UnofficialEventId: "ue-o1"}).Error)
	require.NoError(t, db.Create(&model.Match{ID: "mat-o1", CreatedAt: now, UpdatedAt: now, RecordId: "rec-o1", UserId: otherUid}).Error)
	require.NoError(t, db.Create(&model.Game{ID: "gam-o1", CreatedAt: now, UpdatedAt: now, MatchId: "mat-o1", UserId: otherUid}).Error)

	ctx := context.Background()
	require.NoError(t, NewRecord(db, slog.Default()).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewDeck(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewDeckCode(db).DeleteByUserId(ctx, withdrawUid))

	// alive は論理削除されずに残っている行のIDを返す
	alive := func(table string) []string {
		var ids []string
		require.NoError(t, db.Table(table).Where("deleted_at IS NULL").Order("id ASC").Pluck("id", &ids).Error)
		return ids
	}

	t.Run("正常系_退会者の記録と対戦結果と対局が削除される", func(t *testing.T) {
		require.Equal(t, []string{"rec-o1"}, alive("records"))
		require.Equal(t, []string{"mat-o1"}, alive("matches"))
		require.Equal(t, []string{"gam-o1"}, alive("games"))
	})

	t.Run("正常系_記録が参照していた自由形式イベントも削除される", func(t *testing.T) {
		require.Equal(t, []string{"ue-o1"}, alive("unofficial_events"))
	})

	t.Run("正常系_アーカイブ済みも含め退会者のデッキが削除される", func(t *testing.T) {
		require.Equal(t, []string{"deck-o1"}, alive("decks"))
	})

	t.Run("正常系_退会者が作成したものと退会者のデッキに紐づくものが削除される", func(t *testing.T) {
		// dc-w1/w2: 本人のデッキかつ本人作成、dc-w3: 他人のデッキだが本人作成、
		// dc-o1: 他人が作成したが本人のデッキに紐づく。いずれも残らない。
		require.Empty(t, alive("deck_codes"))
	})
}
