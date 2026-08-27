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

	t.Run("正常系_更新してもcreated_atは変わらない", func(t *testing.T) {
		before, err := r.FindById(context.Background(), id)
		require.NoError(t, err)
		require.False(t, before.CreatedAt.IsZero())

		// 取得した内容をそのまま引き継いで更新する(自由形式イベントの編集と同じ流れ)。
		// created_atを渡さずSaveするとGORMが全カラムを書き戻して更新時刻で潰れるため、
		// 実DBでも保持されることを確認する。
		date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
		updated := entity.NewUnofficialEvent(id, uid, "身内対戦会", date)
		updated.CreatedAt = before.CreatedAt

		require.NoError(t, r.Save(context.Background(), updated))

		ret, err := r.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, "身内対戦会", ret.Title)
		require.Equal(t, date.Format(time.DateOnly), ret.Date.Format(time.DateOnly))
		require.WithinDuration(t, before.CreatedAt, ret.CreatedAt, time.Second)
	})

	t.Run("正常系_削除したイベントは取得できない", func(t *testing.T) {
		deletedId := "01HD7Y3K8D6FDHMHTZ2GT41TU2"
		event := entity.NewUnofficialEvent(deletedId, uid, "削除する自主大会", time.Now().Local())
		require.NoError(t, r.Save(context.Background(), event))

		require.NoError(t, r.Delete(context.Background(), deletedId))

		_, err := r.FindById(context.Background(), deletedId)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundになる", func(t *testing.T) {
		_, err := r.FindById(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TU9")

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})
}

// Tonamel大会情報のキャッシュ(tonamel_events)への保存・バッチ参照を実DBで確認する。
// sqlmockではSQL文字列しか見られず、schema.sqlとの整合(TEXT型・upsert)は保証されない。
func TestIntegrationTonamelEventStore(t *testing.T) {
	db := setupIntegrationDB(t, "tonamel_events")
	r := NewTonamelEventStore(db)
	ctx := context.Background()

	t.Run("正常系_保存した大会をFindByIdsでまとめて取得できる", func(t *testing.T) {
		require.NoError(t, r.Save(ctx, entity.NewTonamelEvent("61ozP", "大会A", "説明A", "https://example.com/a.png")))
		require.NoError(t, r.Save(ctx, entity.NewTonamelEvent("OakZc", "大会B", "", "")))

		// 存在しないIDを混ぜても、あるものだけ返る(無いものはエラーにしない)
		ret, err := r.FindByIds(ctx, []string{"61ozP", "OakZc", "nothere"})

		require.NoError(t, err)
		require.Len(t, ret, 2)

		byId := map[string]*entity.TonamelEvent{}
		for _, e := range ret {
			byId[e.ID] = e
		}
		require.Equal(t, "大会A", byId["61ozP"].Title)
		require.Equal(t, "説明A", byId["61ozP"].Description)
		require.Equal(t, "https://example.com/a.png", byId["61ozP"].Image)
		require.Equal(t, "大会B", byId["OakZc"].Title)
	})

	t.Run("正常系_同じIDのSaveは上書きになる", func(t *testing.T) {
		require.NoError(t, r.Save(ctx, entity.NewTonamelEvent("61ozP", "大会A改", "説明A改", "https://example.com/a2.png")))

		ret, err := r.FindByIds(ctx, []string{"61ozP"})

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, "大会A改", ret[0].Title)
		require.Equal(t, "説明A改", ret[0].Description)
	})

	t.Run("正常系_空スライスは空を返す", func(t *testing.T) {
		ret, err := r.FindByIds(ctx, []string{})

		require.NoError(t, err)
		require.Empty(t, ret)
	})
}

// 「見る」利用の日次シグナル(user_daily_activities)のupsertを実DBで確認する。
// (user_id, date, category)のPKで1日1行に収れんすること、カテゴリが違えば別行になること、
// 再訪でsignal_countが積まれることは、閲覧WAU・記録経験者の4層分解
// (USER_DAILY_ACTIVITIES_PLAN.md §7)の前提になる。
// sqlmockではSQL文字列しか見られないため、実際に行がどう残るかをここで検証する。
func TestIntegrationUserDailyActivityRepository(t *testing.T) {
	db := setupIntegrationDB(t, "user_daily_activities")
	r := NewUserDailyActivity(db)
	ctx := context.Background()

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	morning := time.Date(2026, 8, 4, 10, 0, 0, 0, time.Local)
	night := time.Date(2026, 8, 4, 21, 30, 0, 0, time.Local)

	// サブテスト間で干渉しないよう、ケースごとに別の日付を使う
	day1 := time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)
	day2 := time.Date(2026, 8, 5, 0, 0, 0, 0, time.Local)
	day3 := time.Date(2026, 8, 6, 0, 0, 0, 0, time.Local)

	countOf := func(t *testing.T, date time.Time) int64 {
		t.Helper()

		var count int64
		require.NoError(t, db.Model(&model.UserDailyActivity{}).
			Where("user_id = ? AND date = ?", uid, date).Count(&count).Error)

		return count
	}

	findOne := func(t *testing.T, date time.Time, category string) model.UserDailyActivity {
		t.Helper()

		var m model.UserDailyActivity
		require.NoError(t, db.Where("user_id = ? AND date = ? AND category = ?", uid, date, category).
			First(&m).Error)

		return m
	}

	t.Run("正常系_同日同カテゴリの再訪は1行のままsignal_countが積まれる", func(t *testing.T) {
		require.NoError(t, r.Touch(ctx, []*entity.UserDailyActivity{
			entity.NewUserDailyActivity(uid, day1, entity.UserDailyActivityCategoryVisit, morning),
		}))
		require.NoError(t, r.Touch(ctx, []*entity.UserDailyActivity{
			entity.NewUserDailyActivity(uid, day1, entity.UserDailyActivityCategoryVisit, night),
		}))

		require.Equal(t, int64(1), countOf(t, day1))

		m := findOne(t, day1, entity.UserDailyActivityCategoryVisit)
		require.Equal(t, 2, m.SignalCount)
		// updated_atは最後のシグナル時刻へ更新する(通知→来訪の遅延測定に使うため)
		require.Equal(t, night, m.UpdatedAt.Local())
	})

	t.Run("正常系_同日でもカテゴリが違えば別行になる", func(t *testing.T) {
		require.NoError(t, r.Touch(ctx, []*entity.UserDailyActivity{
			entity.NewUserDailyActivity(uid, day2, entity.UserDailyActivityCategoryVisit, morning),
			entity.NewUserDailyActivity(uid, day2, entity.UserDailyActivityCategoryReview, morning),
		}))

		require.Equal(t, int64(2), countOf(t, day2))
		require.Equal(t, 1, findOne(t, day2, entity.UserDailyActivityCategoryVisit).SignalCount)
		require.Equal(t, 1, findOne(t, day2, entity.UserDailyActivityCategoryReview).SignalCount)
	})

	t.Run("正常系_reviewだけが届いても行は作られる", func(t *testing.T) {
		// 「その日開いたか」は行の存在で判定するため(visit行の有無では判定しない)、
		// visitのビーコンが落ちてreviewだけ届いた場合も訪問として数えられる必要がある。
		require.NoError(t, r.Touch(ctx, []*entity.UserDailyActivity{
			entity.NewUserDailyActivity(uid, day3, entity.UserDailyActivityCategoryReview, morning),
		}))

		require.Equal(t, int64(1), countOf(t, day3))
		require.Equal(t, 1, findOne(t, day3, entity.UserDailyActivityCategoryReview).SignalCount)
	})
}

// 退会時の一括削除(DeleteByUserId)が、退会者の関連データを漏れなく消し、かつ
// 他ユーザのデータを巻き込まないことを実DBで確認する。
// 一括削除はサブクエリで対象を絞るため、条件を1つ間違えると他人のデータまで
// 消えうる。sqlmockではSQL文字列しか見られないので、ここで実際の結果を検証する。
func TestIntegrationDeleteByUserId(t *testing.T) {
	db := setupIntegrationDB(t,
		"match_pokemon_sprites", "deck_pokemon_sprites",
		"match_tags", "record_tags", "deck_code_tags", "deck_tags", "tags",
		"user_favorite_decks", "user_streaks", "user_daily_activities",
		"user_badges", "user_environment_badges", "notifications", "users_players",
		"games", "matches", "records", "deck_codes", "decks", "unofficial_events",
		"pokemon_sprites",
	)

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

	// どの記録からも参照されていない、作りかけの自由形式イベント(user_id からしか辿れない)
	require.NoError(t, db.Create(&model.UnofficialEvent{ID: "ue-w2", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, Title: "使わなかった自主大会", Date: now}).Error)

	// タグ(退会者・他人・全ユーザー共通のプリセット)と、その付与
	require.NoError(t, db.Create(&model.Tag{ID: "tag-w1", CreatedAt: now, UpdatedAt: now, UserId: withdrawUid, Name: "退会者のタグ"}).Error)
	require.NoError(t, db.Create(&model.Tag{ID: "tag-o1", CreatedAt: now, UpdatedAt: now, UserId: otherUid, Name: "他人のタグ"}).Error)
	require.NoError(t, db.Create(&model.Tag{ID: "tag-p1", CreatedAt: now, UpdatedAt: now, UserId: "", Name: "🥇 優勝", Color: "#FFB900", PresetFlg: true, PresetCategory: entity.TagPresetCategoryPlacement, TextColor: "#461901"}).Error)

	// スプライト(中間テーブルのFK先)
	require.NoError(t, db.Exec(`INSERT INTO pokemon_sprites (id, name) VALUES ('0025', 'ピカチュウ')`).Error)

	// --- 巻き込まれてはいけない他人のデータ
	require.NoError(t, db.Create(&model.UnofficialEvent{ID: "ue-o1", CreatedAt: now, UpdatedAt: now, UserId: otherUid, Title: "他人の自主大会", Date: now}).Error)
	require.NoError(t, db.Create(&model.Record{ID: "rec-o1", CreatedAt: now, UpdatedAt: now, UserId: otherUid, DeckId: "deck-o1", EventDate: now, UnofficialEventId: "ue-o1"}).Error)
	require.NoError(t, db.Create(&model.Match{ID: "mat-o1", CreatedAt: now, UpdatedAt: now, RecordId: "rec-o1", UserId: otherUid}).Error)
	require.NoError(t, db.Create(&model.Game{ID: "gam-o1", CreatedAt: now, UpdatedAt: now, MatchId: "mat-o1", UserId: otherUid}).Error)
	require.NoError(t, db.Create(&model.DeckCode{ID: "dc-o2", CreatedAt: now, UpdatedAt: now, UserId: otherUid, DeckId: "deck-o1", Code: "eeee"}).Error)

	// 中間テーブル(退会者ぶんと他人ぶんを対にして入れる)
	require.NoError(t, db.Create(&model.DeckTag{DeckId: "deck-w1", TagId: "tag-w1"}).Error)
	require.NoError(t, db.Create(&model.DeckTag{DeckId: "deck-o1", TagId: "tag-o1"}).Error)
	require.NoError(t, db.Create(&model.DeckCodeTag{DeckCodeId: "dc-w1", TagId: "tag-w1"}).Error)
	require.NoError(t, db.Create(&model.DeckCodeTag{DeckCodeId: "dc-o2", TagId: "tag-o1"}).Error)
	require.NoError(t, db.Create(&model.RecordTag{RecordId: "rec-w1", TagId: "tag-w1"}).Error)
	require.NoError(t, db.Create(&model.RecordTag{RecordId: "rec-o1", TagId: "tag-o1"}).Error)
	require.NoError(t, db.Create(&model.MatchTag{MatchId: "mat-w1", TagId: "tag-w1"}).Error)
	require.NoError(t, db.Create(&model.MatchTag{MatchId: "mat-o1", TagId: "tag-o1"}).Error)
	require.NoError(t, db.Create(&model.DeckPokemonSprite{DeckId: "deck-w1", Position: 1, PokemonSpriteId: "0025"}).Error)
	require.NoError(t, db.Create(&model.DeckPokemonSprite{DeckId: "deck-o1", Position: 1, PokemonSpriteId: "0025"}).Error)
	require.NoError(t, db.Create(&model.MatchPokemonSprite{MatchId: "mat-w1", Position: 1, PokemonSpriteId: "0025"}).Error)
	require.NoError(t, db.Create(&model.MatchPokemonSprite{MatchId: "mat-o1", Position: 1, PokemonSpriteId: "0025"}).Error)

	// お気に入り: 退会者が他人のデッキに付けたもの / 他人が退会者のデッキに付けたもの /
	// 他人が他人のデッキに付けたもの(これだけ残る)
	require.NoError(t, db.Create(&model.UserFavoriteDeck{UserId: withdrawUid, DeckId: "deck-o1", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&model.UserFavoriteDeck{UserId: otherUid, DeckId: "deck-w1", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&model.UserFavoriteDeck{UserId: otherUid, DeckId: "deck-o1", CreatedAt: now}).Error)

	// ユーザ単位の付随データ(いずれも論理削除を持たない)
	for _, uid := range []string{withdrawUid, otherUid} {
		require.NoError(t, db.Create(&model.UserStreak{UserId: uid, LastRecordedWeek: now, UpdatedAt: now}).Error)
		require.NoError(t, db.Create(&model.UserDailyActivity{UserId: uid, Date: now, Category: "visit", SignalCount: 1, UpdatedAt: now}).Error)
		require.NoError(t, db.Create(&model.UserBadge{ID: "ub-" + uid[:4], CreatedAt: now, UserId: uid, BadgeDefinitionId: "onboarding-00", AchievedAt: now}).Error)
		require.NoError(t, db.Create(&model.UserEnvironmentBadge{UserId: uid, EnvironmentId: "m6a", AchievedAt: now, CreatedAt: now}).Error)
		require.NoError(t, db.Create(&model.Notification{ID: "ntf-" + uid[:4], CreatedAt: now, UserId: uid, Category: "badge", Title: "t", Body: "b"}).Error)
		require.NoError(t, db.Create(&model.UserPlayer{ID: "up-" + uid[:4], CreatedAt: now, UpdatedAt: now, UserId: uid, PlayerId: "1234567890123456"}).Error)
	}

	// 退会処理(usecase.User.Delete)が呼ぶのと同じ順序・同じリポジトリで消す。
	// 「どのリポジトリを呼ぶか」は usecase のユニットテストが担保する。
	ctx := context.Background()
	require.NoError(t, NewRecord(db, slog.Default()).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewDeck(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewDeckCode(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewTag(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewUserFavoriteDeck(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewUserStreak(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewUserDailyActivity(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewUserBadge(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewUserEnvironmentBadge(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewNotification(db).DeleteByUserId(ctx, withdrawUid))
	require.NoError(t, NewUserPlayer(db).DeleteByUserId(ctx, withdrawUid))

	// alive は論理削除されずに残っている行のIDを返す
	alive := func(table string) []string {
		var ids []string
		require.NoError(t, db.Table(table).Where("deleted_at IS NULL").Order("id ASC").Pluck("id", &ids).Error)
		return ids
	}

	// remaining は table に残っている行数を返す(where で対象を絞る)
	remaining := func(table string, where string, args ...any) int64 {
		var n int64
		require.NoError(t, db.Table(table).Where(where, args...).Count(&n).Error)
		return n
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
		// dc-o2 は他人のデッキに他人が作ったものなので残る。
		require.Equal(t, []string{"dc-o2"}, alive("deck_codes"))
	})

	t.Run("正常系_どの記録からも参照されていない自由形式イベントも削除される", func(t *testing.T) {
		// ue-w2 は作りっぱなしで record から参照されていない。user_id からしか辿れない。
		require.Equal(t, []string{"ue-o1"}, alive("unofficial_events"))
	})

	t.Run("正常系_退会者のタグが削除されプリセットは残る", func(t *testing.T) {
		// プリセット(user_id='')は全ユーザー共通なので、退会で消してはいけない。
		require.Equal(t, []string{"tag-o1", "tag-p1"}, alive("tags"))
	})

	t.Run("正常系_中間テーブルの行も残らない", func(t *testing.T) {
		// 中間テーブルは論理削除を持たないため、消し残すと参照先の無い行になる。
		require.Zero(t, remaining("deck_tags", "tag_id = ? OR deck_id IN ?", "tag-w1", []string{"deck-w1", "deck-w2"}))
		require.Zero(t, remaining("deck_code_tags", "tag_id = ? OR deck_code_id IN ?", "tag-w1", []string{"dc-w1", "dc-w2", "dc-w3", "dc-o1"}))
		require.Zero(t, remaining("record_tags", "tag_id = ? OR record_id IN ?", "tag-w1", []string{"rec-w1", "rec-w2"}))
		require.Zero(t, remaining("match_tags", "tag_id = ? OR match_id IN ?", "tag-w1", []string{"mat-w1", "mat-w2"}))
		require.Zero(t, remaining("deck_pokemon_sprites", "deck_id IN ?", []string{"deck-w1", "deck-w2"}))
		require.Zero(t, remaining("match_pokemon_sprites", "match_id IN ?", []string{"mat-w1", "mat-w2"}))

		// 他人のぶんは1行ずつ残っている
		require.Equal(t, int64(1), remaining("deck_tags", "1 = 1"))
		require.Equal(t, int64(1), remaining("deck_code_tags", "1 = 1"))
		require.Equal(t, int64(1), remaining("record_tags", "1 = 1"))
		require.Equal(t, int64(1), remaining("match_tags", "1 = 1"))
		require.Equal(t, int64(1), remaining("deck_pokemon_sprites", "1 = 1"))
		require.Equal(t, int64(1), remaining("match_pokemon_sprites", "1 = 1"))
	})

	t.Run("正常系_お気に入りは退会者が付けたものも退会者のデッキに付いたものも消える", func(t *testing.T) {
		// 残るのは「他人が他人のデッキに付けたもの」だけ。
		require.Zero(t, remaining("user_favorite_decks", "user_id = ?", withdrawUid))
		require.Equal(t, int64(1), remaining("user_favorite_decks", "1 = 1"))
	})

	t.Run("正常系_ユーザ単位の付随データが残らない", func(t *testing.T) {
		for _, table := range []string{
			"user_streaks", "user_daily_activities", "user_badges",
			"user_environment_badges", "notifications",
		} {
			require.Zero(t, remaining(table, "user_id = ?", withdrawUid), "%s に退会者の行が残っている", table)
			require.Equal(t, int64(1), remaining(table, "user_id = ?", otherUid), "%s の他人の行が巻き込まれている", table)
		}

		require.Zero(t, remaining("users_players", "user_id = ? AND deleted_at IS NULL", withdrawUid))
		require.Equal(t, int64(1), remaining("users_players", "user_id = ? AND deleted_at IS NULL", otherUid))
	})
}

// お気に入りは decks の列ではなく user_favorite_decks で管理している。
// JOIN の当たり方・並び順・削除の連鎖は sqlmock では確かめられないため、実DBで見る。
func TestIntegrationUserFavoriteDeckRepository(t *testing.T) {
	db := setupIntegrationDB(t, "user_favorite_decks", "deck_codes", "decks")

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	now := time.Now().Local().Truncate(time.Microsecond)
	// 「古いデッキをお気に入りにすると、新しいデッキより前に出る」ことを見たいので、
	// お気に入りにするデッキ(deck-old)の作成日時を最も古くしておく。
	require.NoError(t, db.Create(&model.Deck{ID: "deck-old", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now, UserId: uid, Name: "古いデッキ"}).Error)
	require.NoError(t, db.Create(&model.Deck{ID: "deck-new", CreatedAt: now, UpdatedAt: now, UserId: uid, Name: "新しいデッキ"}).Error)

	ctx := context.Background()
	r := NewUserFavoriteDeck(db)
	deckRepository := NewDeck(db)

	favoritedAt := now.Add(-time.Minute)

	t.Run("正常系_お気に入りを追加して取得できる", func(t *testing.T) {
		require.NoError(t, r.Create(ctx, entity.NewUserFavoriteDeck(uid, "deck-old", favoritedAt)))

		favorites, err := r.FindByUserId(ctx, uid)

		require.NoError(t, err)
		require.Len(t, favorites, 1)
		require.Equal(t, "deck-old", favorites[0].DeckId)
		// DBから戻る時刻は Location が接続のTZ(Asia/Tokyo)になるため、
		// 表現ではなく指している瞬間で比べる。
		require.True(t, favoritedAt.Equal(favorites[0].CreatedAt))
	})

	t.Run("正常系_お気に入りのデッキが作成日時によらず一覧の先頭に来る", func(t *testing.T) {
		decks, err := deckRepository.FindAll(ctx, uid)

		require.NoError(t, err)
		require.Len(t, decks, 2)
		require.Equal(t, "deck-old", decks[0].ID)
		require.True(t, favoritedAt.Equal(decks[0].FavoritedAt))
		// お気に入りでないデッキは FavoritedAt がゼロ値のまま
		require.Equal(t, "deck-new", decks[1].ID)
		require.True(t, decks[1].FavoritedAt.IsZero())
	})

	t.Run("正常系_デッキを削除するとお気に入りも消える", func(t *testing.T) {
		require.NoError(t, deckRepository.Delete(ctx, "deck-old"))

		favorites, err := r.FindByUserId(ctx, uid)

		require.NoError(t, err)
		require.Empty(t, favorites)
	})

	t.Run("正常系_解除したお気に入りは一覧に残らない", func(t *testing.T) {
		require.NoError(t, r.Create(ctx, entity.NewUserFavoriteDeck(uid, "deck-new", favoritedAt)))
		require.NoError(t, r.Delete(ctx, uid, "deck-new"))

		favorites, err := r.FindByUserId(ctx, uid)

		require.NoError(t, err)
		require.Empty(t, favorites)
	})
}

// タグは tags マスタと deck_tags / deck_code_tags 中間テーブルにまたがる。
// 生SQLの INSERT/DELETE(replaceTags)と JOIN 読み出し(findTagsBy*)、削除時の連鎖は
// sqlmock では確かめられないため、実DBで一連の往復を見る。
func TestIntegrationTagRepository(t *testing.T) {
	db := setupIntegrationDB(t, "deck_code_tags", "deck_tags", "tags", "deck_codes", "decks")

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	const otherUid = "KBp7roRDZobZg1t0OPzFR1kvLeO2"

	now := time.Now().Local().Truncate(time.Microsecond)

	// 付与先(FK制約があるため先に用意する)
	require.NoError(t, db.Create(&model.Deck{ID: "deck-t1", CreatedAt: now, UpdatedAt: now, UserId: uid, Name: "デッキ"}).Error)
	require.NoError(t, db.Create(&model.DeckCode{ID: "dc-t1", CreatedAt: now, UpdatedAt: now, UserId: uid, DeckId: "deck-t1", Code: "aaaa"}).Error)

	ctx := context.Background()
	r := NewTag(db)
	deckRepository := NewDeck(db)
	deckCodeRepository := NewDeckCode(db)

	t.Run("正常系_タグを保存して一覧と名前引きで取得できる", func(t *testing.T) {
		require.NoError(t, r.Save(ctx, entity.NewTag("tag-1", now, now, uid, "アグロ", "#ff0000", false, "", "")))
		require.NoError(t, r.Save(ctx, entity.NewTag("tag-2", now.Add(time.Second), now, uid, "コントロール", "", false, "", "")))
		// 他人のタグは混ざってはいけない
		require.NoError(t, r.Save(ctx, entity.NewTag("tag-x", now, now, otherUid, "アグロ", "", false, "", "")))

		tags, err := r.FindByUserId(ctx, uid)
		require.NoError(t, err)
		require.Len(t, tags, 2)
		// created_at の降順
		require.Equal(t, "tag-2", tags[0].ID)
		require.Equal(t, "tag-1", tags[1].ID)
		require.Equal(t, "#ff0000", tags[1].Color)

		got, err := r.FindByUserIdAndName(ctx, uid, "アグロ")
		require.NoError(t, err)
		require.Equal(t, "tag-1", got.ID)

		_, err = r.FindByUserIdAndName(ctx, uid, "存在しない")
		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})

	t.Run("正常系_FindAttachableByIdsは付与できるタグだけ返す", func(t *testing.T) {
		// 自分のタグ + 存在しないID + 他人のタグ を混ぜても、自分のものだけ返る
		tags, err := r.FindAttachableByIds(ctx, []string{"tag-1", "tag-2", "not-exist", "tag-x"}, uid)
		require.NoError(t, err)
		require.Len(t, tags, 2)

		// 他人のIDだけを自分のuidで引いても空
		tags, err = r.FindAttachableByIds(ctx, []string{"tag-x"}, uid)
		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("正常系_ReplaceDeckTagsで付与を差分更新しデッキ読み出しに載る", func(t *testing.T) {
		require.NoError(t, r.ReplaceDeckTags(ctx, "deck-t1", []string{"tag-1", "tag-2", "tag-1"})) // 重複は無視される

		deck, err := deckRepository.FindById(ctx, "deck-t1")
		require.NoError(t, err)
		require.Len(t, deck.Tags, 2)
		// 付与した順(tag-1, tag-2)で並ぶ。tag-1 の created_at は tag-2 より前なので、
		// これは created_at 降順とは逆であり、付与順(position)で並んでいることを保証する。
		require.Equal(t, "tag-1", deck.Tags[0].ID)
		require.Equal(t, "tag-2", deck.Tags[1].ID)

		// DBに保存される position は1始まり(0始まりではない)であることを実値で確認する。
		var positions []int
		require.NoError(t, db.Raw(
			"SELECT position FROM deck_tags WHERE deck_id = ? ORDER BY position", "deck-t1",
		).Scan(&positions).Error)
		require.Equal(t, []int{1, 2}, positions)

		// 付与し直すと順序も入れ替わる(tag-2 を先に付ける)
		require.NoError(t, r.ReplaceDeckTags(ctx, "deck-t1", []string{"tag-2", "tag-1"}))
		deck, err = deckRepository.FindById(ctx, "deck-t1")
		require.NoError(t, err)
		require.Len(t, deck.Tags, 2)
		require.Equal(t, "tag-2", deck.Tags[0].ID)
		require.Equal(t, "tag-1", deck.Tags[1].ID)

		// 集合を tag-2 だけに絞る
		require.NoError(t, r.ReplaceDeckTags(ctx, "deck-t1", []string{"tag-2"}))
		deck, err = deckRepository.FindById(ctx, "deck-t1")
		require.NoError(t, err)
		require.Len(t, deck.Tags, 1)
		require.Equal(t, "tag-2", deck.Tags[0].ID)

		// 空にすると付与が消える
		require.NoError(t, r.ReplaceDeckTags(ctx, "deck-t1", []string{}))
		deck, err = deckRepository.FindById(ctx, "deck-t1")
		require.NoError(t, err)
		require.Empty(t, deck.Tags)
	})

	t.Run("正常系_ReplaceDeckCodeTagsで付与しデッキコード読み出しに載る", func(t *testing.T) {
		require.NoError(t, r.ReplaceDeckCodeTags(ctx, "dc-t1", []string{"tag-1"}))

		deckcode, err := deckCodeRepository.FindById(ctx, "dc-t1")
		require.NoError(t, err)
		require.Len(t, deckcode.Tags, 1)
		require.Equal(t, "tag-1", deckcode.Tags[0].ID)

		// デッキ読み出しでも、最新バージョン(latest_deck_code)のタグが載る
		// (新バージョン作成時のタグ継承で使う。dc-t1 は deck-t1 唯一=最新のデッキコード)。
		deck, err := deckRepository.FindById(ctx, "deck-t1")
		require.NoError(t, err)
		require.NotNil(t, deck.LatestDeckCode)
		require.Equal(t, "dc-t1", deck.LatestDeckCode.ID)
		require.Len(t, deck.LatestDeckCode.Tags, 1)
		require.Equal(t, "tag-1", deck.LatestDeckCode.Tags[0].ID)
	})

	t.Run("正常系_タグ削除で本体は論理削除され中間テーブルの付与も消える", func(t *testing.T) {
		// deck-t1 に tag-1 を付け直してから tag-1 を消す
		require.NoError(t, r.ReplaceDeckTags(ctx, "deck-t1", []string{"tag-1"}))

		require.NoError(t, r.Delete(ctx, "tag-1"))

		// 論理削除されて名前引き・ID引きの対象から外れる
		_, err := r.FindByUserIdAndName(ctx, uid, "アグロ")
		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		_, err = r.FindById(ctx, "tag-1")
		require.ErrorIs(t, err, apperror.ErrRecordNotFound)

		// 付与(deck_tags / deck_code_tags)も消えている
		deck, err := deckRepository.FindById(ctx, "deck-t1")
		require.NoError(t, err)
		require.Empty(t, deck.Tags)

		deckcode, err := deckCodeRepository.FindById(ctx, "dc-t1")
		require.NoError(t, err)
		require.Empty(t, deckcode.Tags)

		// 一覧には残った tag-2 だけが返る
		tags, err := r.FindByUserId(ctx, uid)
		require.NoError(t, err)
		require.Len(t, tags, 1)
		require.Equal(t, "tag-2", tags[0].ID)
	})

	t.Run("正常系_プリセットタグは一覧と別枠で扱われ誰でも付与できる", func(t *testing.T) {
		// プリセット(全ユーザー共通): user_id='' / preset_flg=true
		require.NoError(t, r.Save(ctx, entity.NewTag("preset-1", now, now, "", "マスターボール", "#FF007F", true, entity.TagPresetCategoryAceSpec, "")))

		// ユーザーの一覧(FindByUserId)にはプリセットは出ない
		userTags, err := r.FindByUserId(ctx, uid)
		require.NoError(t, err)
		require.Len(t, userTags, 1)
		require.Equal(t, "tag-2", userTags[0].ID)

		// プリセット一覧には出る
		presets, err := r.FindPresets(ctx, entity.TagPresetCategoryAceSpec)
		require.NoError(t, err)
		require.Len(t, presets, 1)
		require.Equal(t, "preset-1", presets[0].ID)
		require.True(t, presets[0].PresetFlg)

		// 付与可否: 自分のタグ + プリセットは付与できるが、他人のタグは付与できない
		attachable, err := r.FindAttachableByIds(ctx, []string{"tag-2", "preset-1", "tag-x"}, uid)
		require.NoError(t, err)
		require.Len(t, attachable, 2)

		// ユーザーが自分のデッキにプリセットを付与でき、読み出しに載る(preset_flgも復元される)
		require.NoError(t, r.ReplaceDeckTags(ctx, "deck-t1", []string{"preset-1"}))
		deck, err := deckRepository.FindById(ctx, "deck-t1")
		require.NoError(t, err)
		require.Len(t, deck.Tags, 1)
		require.Equal(t, "preset-1", deck.Tags[0].ID)
		require.True(t, deck.Tags[0].PresetFlg)
	})
}

// FindPresets はプリセットを id 昇順(=作成順=ACE SPEC の card id 昇順)で返す。
// name 昇順ではないことを、id の昇順と name の昇順が逆になるデータで確認する。
func TestIntegrationTagPresetOrder(t *testing.T) {
	db := setupIntegrationDB(t, "tags")
	ctx := context.Background()
	r := NewTag(db)

	now := time.Now().Local().Truncate(time.Microsecond)

	// id "aaa..."(小) の name は "ゼ..."(後方)、id "zzz..."(大) の name は "ア..."(前方)。
	// FindPresets が id 昇順なら [小id, 大id]、name 昇順なら [大id, 小id] になる。
	smallID := "aaaaaaaaaaaaaaaaaaaaaaaaaa"
	largeID := "zzzzzzzzzzzzzzzzzzzzzzzzzz"
	require.NoError(t, r.Save(ctx, entity.NewTag(smallID, now, now, "", "ゼットプリセット", "#FF007F", true, entity.TagPresetCategoryAceSpec, "")))
	require.NoError(t, r.Save(ctx, entity.NewTag(largeID, now, now, "", "アループリセット", "#FF007F", true, entity.TagPresetCategoryAceSpec, "")))

	presets, err := r.FindPresets(ctx, entity.TagPresetCategoryAceSpec)
	require.NoError(t, err)
	require.Len(t, presets, 2)
	// id 昇順(name 昇順の逆)で返ることを確認する。
	require.Equal(t, smallID, presets[0].ID)
	require.Equal(t, largeID, presets[1].ID)
}

// 記録(record)へのタグ付与。record_tags の生SQL(replaceTags)・JOIN読み出し
// (findTagsByRecordIds)・タグ削除時の連鎖を実DBで確認する。
func TestIntegrationRecordTagRepository(t *testing.T) {
	db := setupIntegrationDB(t, "record_tags", "records", "tags")

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	now := time.Now().Local().Truncate(time.Microsecond)

	// 付与先(FK制約: record_tags.record_id -> records.id)
	require.NoError(t, db.Create(&model.Record{ID: "rec-t1", CreatedAt: now, UpdatedAt: now, UserId: uid, EventDate: now, RegulationId: entity.RegulationIdStandard}).Error)

	ctx := context.Background()
	r := NewTag(db)
	recordRepository := NewRecord(db, slog.Default())

	require.NoError(t, r.Save(ctx, entity.NewTag("rtag-1", now, now, uid, "調整中", "", false, "", "")))
	require.NoError(t, r.Save(ctx, entity.NewTag("rtag-2", now.Add(time.Second), now, uid, "本番", "", false, "", "")))
	// 記録向けのプリセット(大会順位)。誰でも付与できる。名前の絵文字まで含めて schema.sql と揃える。
	require.NoError(t, r.Save(ctx, entity.NewTag("rtag-p1", now, now, "", "🥇 優勝", "#FFB900", true, entity.TagPresetCategoryPlacement, "#461901")))

	t.Run("正常系_ReplaceRecordTagsで付与を差分更新し記録の読み出しに載る", func(t *testing.T) {
		require.NoError(t, r.ReplaceRecordTags(ctx, "rec-t1", []string{"rtag-1", "rtag-2"}))

		record, err := recordRepository.FindById(ctx, "rec-t1")
		require.NoError(t, err)
		require.Len(t, record.Tags, 2)
		// 付与した順(rtag-1, rtag-2)で並ぶ
		require.Equal(t, "rtag-1", record.Tags[0].ID)
		require.Equal(t, "rtag-2", record.Tags[1].ID)

		// 集合を rtag-2 だけに絞る
		require.NoError(t, r.ReplaceRecordTags(ctx, "rec-t1", []string{"rtag-2"}))
		record, err = recordRepository.FindById(ctx, "rec-t1")
		require.NoError(t, err)
		require.Len(t, record.Tags, 1)
		require.Equal(t, "rtag-2", record.Tags[0].ID)
	})

	t.Run("正常系_一覧取得でも付与タグがまとめて載る", func(t *testing.T) {
		require.NoError(t, r.ReplaceRecordTags(ctx, "rec-t1", []string{"rtag-p1", "rtag-2"}))

		records, err := recordRepository.FindByUserId(ctx, uid, 10, 0, "")
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Len(t, records[0].Tags, 2)
		require.Equal(t, "rtag-p1", records[0].Tags[0].ID)
		require.True(t, records[0].Tags[0].PresetFlg)
		require.Equal(t, entity.TagPresetCategoryPlacement, records[0].Tags[0].PresetCategory)
		// 配色(背景色+文字色)もJOINの読み出しで復元される
		require.Equal(t, "#FFB900", records[0].Tags[0].Color)
		require.Equal(t, "#461901", records[0].Tags[0].TextColor)
	})

	t.Run("正常系_タグ削除で記録への付与も外れる", func(t *testing.T) {
		require.NoError(t, r.ReplaceRecordTags(ctx, "rec-t1", []string{"rtag-1", "rtag-2"}))

		require.NoError(t, r.Delete(ctx, "rtag-1"))

		record, err := recordRepository.FindById(ctx, "rec-t1")
		require.NoError(t, err)
		require.Len(t, record.Tags, 1)
		require.Equal(t, "rtag-2", record.Tags[0].ID)
	})
}

// プリセットは群(preset_category)で絞って取得できる。記録の付与UIに ACE SPEC を
// 出さず、大会順位だけを出すための絞り込み。
func TestIntegrationTagPresetCategory(t *testing.T) {
	db := setupIntegrationDB(t, "tags")
	ctx := context.Background()
	r := NewTag(db)

	now := time.Now().Local().Truncate(time.Microsecond)

	require.NoError(t, r.Save(ctx, entity.NewTag("pc-ace", now, now, "", "マスターボール", "#FF007F", true, entity.TagPresetCategoryAceSpec, "")))
	require.NoError(t, r.Save(ctx, entity.NewTag("pc-place", now, now, "", "🥇 優勝", "#FFB900", true, entity.TagPresetCategoryPlacement, "#461901")))

	aceSpecs, err := r.FindPresets(ctx, entity.TagPresetCategoryAceSpec)
	require.NoError(t, err)
	require.Len(t, aceSpecs, 1)
	require.Equal(t, "pc-ace", aceSpecs[0].ID)

	placements, err := r.FindPresets(ctx, entity.TagPresetCategoryPlacement)
	require.NoError(t, err)
	require.Len(t, placements, 1)
	require.Equal(t, "pc-place", placements[0].ID)

	// 群を指定しなければ全プリセットが返る(既存クライアント互換)
	all, err := r.FindPresets(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 2)
}

// 対戦結果(match)へのタグ付与。match_tags の生SQL(replaceTags)・JOIN読み出し
// (findTagsByMatchIds)・タグ削除時の連鎖を実DBで確認する。
func TestIntegrationMatchTagRepository(t *testing.T) {
	db := setupIntegrationDB(t, "match_tags", "matches", "records", "tags")

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	now := time.Now().Local().Truncate(time.Microsecond)

	// 付与先(FK制約: match_tags.match_id -> matches.id -> records.id)
	require.NoError(t, db.Create(&model.Record{ID: "rec-m1", CreatedAt: now, UpdatedAt: now, UserId: uid, EventDate: now}).Error)
	require.NoError(t, db.Create(&model.Match{ID: "mat-m1", CreatedAt: now, UpdatedAt: now, RecordId: "rec-m1", UserId: uid}).Error)

	ctx := context.Background()
	r := NewTag(db)
	matchRepository := NewMatch(db)

	require.NoError(t, r.Save(ctx, entity.NewTag("mtag-1", now, now, uid, "接戦", "", false, "", "")))
	require.NoError(t, r.Save(ctx, entity.NewTag("mtag-2", now.Add(time.Second), now, uid, "反省", "", false, "", "")))

	t.Run("正常系_ReplaceMatchTagsで付与を差分更新し対戦結果の読み出しに載る", func(t *testing.T) {
		require.NoError(t, r.ReplaceMatchTags(ctx, "mat-m1", []string{"mtag-1", "mtag-2"}))

		match, err := matchRepository.FindById(ctx, "mat-m1")
		require.NoError(t, err)
		require.Len(t, match.Tags, 2)
		// 付与した順(mtag-1, mtag-2)で並ぶ
		require.Equal(t, "mtag-1", match.Tags[0].ID)
		require.Equal(t, "mtag-2", match.Tags[1].ID)

		// 集合を mtag-2 だけに絞る
		require.NoError(t, r.ReplaceMatchTags(ctx, "mat-m1", []string{"mtag-2"}))
		match, err = matchRepository.FindById(ctx, "mat-m1")
		require.NoError(t, err)
		require.Len(t, match.Tags, 1)
		require.Equal(t, "mtag-2", match.Tags[0].ID)
	})

	t.Run("正常系_タグ削除で対戦結果への付与も外れる", func(t *testing.T) {
		require.NoError(t, r.ReplaceMatchTags(ctx, "mat-m1", []string{"mtag-1", "mtag-2"}))

		require.NoError(t, r.Delete(ctx, "mtag-1"))

		match, err := matchRepository.FindById(ctx, "mat-m1")
		require.NoError(t, err)
		require.Len(t, match.Tags, 1)
		require.Equal(t, "mtag-2", match.Tags[0].ID)
	})
}

// 連携済みプレイヤーIDの入賞取得。official_events・shops・prefectures を結合する生SQLと
// db/schema.sql の整合(カラム名・予約語 rank のエスケープ有無)を実DBで確認する。
func TestIntegrationCityleagueResultFindByPlayerId(t *testing.T) {
	db := setupIntegrationDB(t, "cityleague_results", "official_events")

	const playerId = "1234567890"

	// FK制約: cityleague_results -> cityleague_schedules / official_events -> shops -> prefectures。
	// prefectures と cityleague_schedules は db/schema.sql が初期データを持つのでそのまま使う。
	require.NoError(t, db.Exec(
		`INSERT INTO shops (id, name, term, prefecture_id) VALUES (?, ?, ?, ?) ON CONFLICT (id) DO NOTHING`,
		9001, "ポケモンカードステーション・テスト", 1, 13,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO official_events (id, title, address, date, shop_id, shop_name) VALUES (?, ?, ?, ?, ?, ?)`,
		952749, "シティリーグ2026 シーズン4", "東京都", time.Date(2026, 4, 5, 0, 0, 0, 0, time.Local), 9001, "ポケモンカードステーション・テスト",
	).Error)
	// official_events に対応する行が無い入賞も落とさない(LEFT JOIN)ことを確認するための1件
	require.NoError(t, db.Exec(
		`INSERT INTO official_events (id, title, address, date) VALUES (?, ?, ?, ?)`,
		952750, "シティリーグ2026 シーズン4", "北海道", time.Date(2026, 4, 12, 0, 0, 0, 0, time.Local),
	).Error)

	insertResult := func(officialEventId uint, eventDate time.Time, rank uint, point uint, deckCode string, targetPlayerId string) {
		t.Helper()
		require.NoError(t, db.Create(&model.CityleagueResult{
			CityleagueScheduleId: "2026s4",
			OfficialEventId:      officialEventId,
			LeagueType:           1,
			EventDate:            eventDate,
			PlayerId:             targetPlayerId,
			PlayerName:           "テストプレイヤー",
			Rank:                 rank,
			Point:                point,
			DeckCode:             deckCode,
		}).Error)
	}

	insertResult(952749, time.Date(2026, 4, 5, 0, 0, 0, 0, time.Local), 1, 15, "gnnHHn-Vg3aWc-LHNnHH", playerId)
	insertResult(952750, time.Date(2026, 4, 12, 0, 0, 0, 0, time.Local), 3, 12, "xxxYYY-ZZZzzz-AAAbbb", playerId)
	// 他人の入賞(返してはならない)
	insertResult(952749, time.Date(2026, 4, 5, 0, 0, 0, 0, time.Local), 2, 13, "other-deck-code-0000", "9999999999")

	ctx := context.Background()
	r := NewCityleagueResult(db)

	t.Run("正常系_自分の入賞だけを開催イベント情報込みで新しい順に返す", func(t *testing.T) {
		ret, err := r.FindByPlayerId(ctx, playerId, time.Time{}, time.Time{})

		require.NoError(t, err)
		require.Len(t, ret, 2)

		// event_date の降順
		require.Equal(t, uint(952750), ret[0].OfficialEventId)
		require.Equal(t, uint(3), ret[0].Rank)

		require.Equal(t, uint(952749), ret[1].OfficialEventId)
		require.Equal(t, uint(1), ret[1].Rank)
		require.Equal(t, uint(15), ret[1].Point)
		require.Equal(t, "gnnHHn-Vg3aWc-LHNnHH", ret[1].DeckCode)
		require.Equal(t, "シティリーグ2026 シーズン4", ret[1].EventTitle)
		require.Equal(t, "ポケモンカードステーション・テスト", ret[1].ShopName)
		require.Equal(t, "東京都", ret[1].PrefectureName)
		// 開催日(2026-04-05)が属する対戦環境が db/schema.sql の初期データから引けること
		require.NotEmpty(t, ret[1].EnvironmentTitle)
	})

	t.Run("正常系_店舗が紐付かない入賞も落とさない", func(t *testing.T) {
		ret, err := r.FindByPlayerId(ctx, playerId, time.Time{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, uint(952750), ret[0].OfficialEventId)
		require.Empty(t, ret[0].ShopName)
		require.Empty(t, ret[0].PrefectureName)
	})

	t.Run("正常系_シーズン期間の半開区間で絞り込む", func(t *testing.T) {
		// toDate は exclusive のため、4/12 の入賞は範囲外になる
		ret, err := r.FindByPlayerId(ctx, playerId,
			time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 4, 12, 0, 0, 0, 0, time.Local),
		)

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, uint(952749), ret[0].OfficialEventId)
	})

	t.Run("正常系_該当が無ければ空のスライスを返す", func(t *testing.T) {
		ret, err := r.FindByPlayerId(ctx, "0000000000", time.Time{}, time.Time{})

		require.NoError(t, err)
		require.NotNil(t, ret)
		require.Empty(t, ret)
	})
}

// 記録のレギュレーション(records.regulation_id)。FK制約とDEFAULTの効き方は
// sqlmockでは確認できないため、実DBで往復させる。
func TestIntegrationRecordRegulation(t *testing.T) {
	db := setupIntegrationDB(t, "matches", "records")

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	ctx := context.Background()
	r := NewRecord(db, slog.Default())
	now := time.Now().Local().Truncate(time.Microsecond)

	t.Run("正常系_指定したレギュレーションを保存して読み戻せる", func(t *testing.T) {
		record := entity.NewRecord(
			"rec-reg-1", now, 0, "", "", "", uid, "", "", now, false, false,
			entity.RegulationIdHallOfFame, "", "",
		)

		require.NoError(t, r.Save(ctx, record))

		ret, err := r.FindById(ctx, "rec-reg-1")
		require.NoError(t, err)
		require.Equal(t, entity.RegulationIdHallOfFame, ret.RegulationId)
	})

	// 旧クライアントからの記録や、レギュレーションを埋めない書き込み経路が
	// FK違反にならず、DB側のDEFAULTと同じスタンダードで保存されること。
	t.Run("正常系_レギュレーション未設定はスタンダードとして保存される", func(t *testing.T) {
		record := entity.NewRecord(
			"rec-reg-2", now, 0, "", "", "", uid, "", "", now, false, false,
			0, "", "",
		)

		require.NoError(t, r.Save(ctx, record))

		ret, err := r.FindById(ctx, "rec-reg-2")
		require.NoError(t, err)
		require.Equal(t, entity.RegulationIdStandard, ret.RegulationId)
	})

	t.Run("異常系_存在しないレギュレーションはFK制約で弾かれる", func(t *testing.T) {
		record := entity.NewRecord(
			"rec-reg-3", now, 0, "", "", "", uid, "", "", now, false, false,
			999, "", "",
		)

		require.Error(t, r.Save(ctx, record))
	})
}

// 対戦環境分析(週次デッキ使用率)がスタンダードの記録だけを集計すること。
// レギュレーションでの絞り込みはSQLの条件のため、実DBで効いていることを確認する。
func TestIntegrationWeeklyDeckUsageStatRegulation(t *testing.T) {
	db := setupIntegrationDB(t, "match_pokemon_sprites", "matches", "records", "pokemon_sprites")

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 週の半開区間 [月, 翌月) に収まる開催日で記録を作る
	fromDate := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
	toDate := fromDate.AddDate(0, 0, 7)
	eventDate := fromDate.AddDate(0, 0, 1)
	now := time.Now().Local().Truncate(time.Microsecond)

	require.NoError(t, db.Create(&model.PokemonSprite{ID: "0025", Name: "ピカチュウ"}).Error)

	// 相手デッキのスプライトから票が立つよう、記録・対戦・スプライトを組で作る
	createMatchWithSprite := func(recordId string, matchId string, regulationId uint) {
		t.Helper()

		require.NoError(t, db.Create(&model.Record{
			ID: recordId, CreatedAt: now, UpdatedAt: now, UserId: uid,
			EventDate: eventDate, RegulationId: regulationId,
		}).Error)
		require.NoError(t, db.Create(&model.Match{
			ID: matchId, CreatedAt: now, UpdatedAt: now, RecordId: recordId, UserId: uid,
		}).Error)
		require.NoError(t, db.Create(&model.MatchPokemonSprite{
			MatchId: matchId, Position: 1, PokemonSpriteId: "0025",
		}).Error)
	}

	createMatchWithSprite("rec-wk-std", "mat-wk-std", entity.RegulationIdStandard)
	createMatchWithSprite("rec-wk-ext", "mat-wk-ext", entity.RegulationIdExtra)
	createMatchWithSprite("rec-wk-hof", "mat-wk-hof", entity.RegulationIdHallOfFame)

	r := NewWeeklyDeckUsageStat(db)

	stat, err := r.FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate)
	require.NoError(t, err)

	// スタンダードの1マッチぶん(相手側の1票)だけが集計される。
	// エクストラ・殿堂も混ざるなら3票になる。
	require.Equal(t, 1, stat.TotalVotes)
	require.Equal(t, 1, stat.ContributorCount)
}

// 個人の戦績分析(勝率・月次推移・直近N戦・デッキ使用率・相手デッキ分布)を
// レギュレーションで絞り込めること。5つの集計すべてで同じ条件が効くかを実DBで確認する。
func TestIntegrationStatsRegulationFilter(t *testing.T) {
	db := setupIntegrationDB(t, "games", "matches", "records", "decks")

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	const deckId = "deck-reg-filter"

	now := time.Now().Local().Truncate(time.Microsecond)
	eventDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	fromDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	toDate := fromDate.AddDate(0, 1, 0)

	require.NoError(t, db.Create(&model.Deck{
		ID: deckId, CreatedAt: now, UpdatedAt: now, UserId: uid, Name: "テストデッキ",
	}).Error)

	createRecord := func(recordId string, regulationId uint) {
		t.Helper()

		require.NoError(t, db.Create(&model.Record{
			ID: recordId, CreatedAt: now, UpdatedAt: now, UserId: uid, DeckId: deckId,
			EventDate: eventDate, RegulationId: regulationId,
		}).Error)
	}
	createMatch := func(matchId string, recordId string, victory bool, opponentsDeckInfo string) {
		t.Helper()

		require.NoError(t, db.Create(&model.Match{
			ID: matchId, CreatedAt: now, UpdatedAt: now, RecordId: recordId, UserId: uid,
			DeckId: deckId, VictoryFlg: victory, OpponentsDeckInfo: opponentsDeckInfo,
		}).Error)
	}

	// スタンダード: 1勝1敗。エクストラ: 1勝(絞り込みで落ちるべき分)。
	createRecord("rec-std", entity.RegulationIdStandard)
	createMatch("mat-std-1", "rec-std", true, "リザードンex")
	createMatch("mat-std-2", "rec-std", false, "リザードンex")

	createRecord("rec-ext", entity.RegulationIdExtra)
	createMatch("mat-ext-1", "rec-ext", true, "ミュウツーGX")

	ctx := context.Background()
	standard := entity.RegulationIdStandard

	t.Run("正常系_戦績はスタンダードの対戦だけを数える", func(t *testing.T) {
		stat, err := NewUserStat(db).FindUserStat(ctx, uid, fromDate, toDate, standard)

		require.NoError(t, err)
		require.Equal(t, 1, stat.TotalRecords)
		require.Equal(t, 2, stat.TotalMatches)
		require.Equal(t, 1, stat.Wins)
		require.Equal(t, 1, stat.Losses)
	})

	t.Run("正常系_月毎の勝率推移はスタンダードの対戦だけを数える", func(t *testing.T) {
		history, err := NewUserStatHistory(db).FindUserStatHistory(ctx, uid, fromDate, toDate, "", standard)

		require.NoError(t, err)
		require.Len(t, history, 1)
		require.Equal(t, 2, history[0].TotalMatches)
		require.Equal(t, 1, history[0].Wins)
	})

	t.Run("正常系_直近N戦はスタンダードの対戦だけを返す", func(t *testing.T) {
		matches, err := NewUserStatRecent(db).FindRecentMatches(ctx, uid, 10, "", standard)

		require.NoError(t, err)
		require.Len(t, matches, 2)
	})

	t.Run("正常系_デッキ使用率はスタンダードの対戦だけを数える", func(t *testing.T) {
		stat, err := NewDeckUsageStat(db).FindDeckUsageStat(ctx, uid, fromDate, toDate, standard)

		require.NoError(t, err)
		require.Len(t, stat.Decks, 1)
		require.Equal(t, deckId, stat.Decks[0].DeckId)
		require.Equal(t, 2, stat.Decks[0].Count)
	})

	t.Run("正常系_相手デッキ分布はスタンダードの対戦だけを数える", func(t *testing.T) {
		stat, err := NewOpponentDeckUsageStat(db).FindOpponentDeckUsageStat(ctx, uid, fromDate, toDate, "", standard)

		require.NoError(t, err)
		require.Equal(t, 2, stat.TotalMatches)
		// エクストラの対戦相手(ミュウツーGX)は含まれない
		for _, deck := range stat.Decks {
			require.NotEqual(t, "ミュウツーGX", deck.DeckInfo)
		}
	})

	t.Run("正常系_未指定なら全レギュレーションを数える", func(t *testing.T) {
		stat, err := NewUserStat(db).FindUserStat(ctx, uid, fromDate, toDate, 0)

		require.NoError(t, err)
		require.Equal(t, 2, stat.TotalRecords)
		require.Equal(t, 3, stat.TotalMatches)
	})
}
