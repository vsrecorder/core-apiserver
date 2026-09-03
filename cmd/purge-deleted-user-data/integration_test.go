package main

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// テスト用のユーザID。実データと衝突しないよう、ULIDでもFirebase UIDでもない形にしてある。
const (
	testTargetUserId = "test_purge_target_uid"
	testOtherUserId  = "test_purge_other_uid"
)

// testId は26文字(ULIDと同じ長さ)のテスト用ID。ID列は VARCHAR(26) のため長さを揃える。
func testId(suffix string) string {
	const prefix = "01TESTPURGE"

	return prefix + strings.Repeat("0", 26-len(prefix)-len(suffix)) + suffix
}

// 物理削除はFK制約に阻まれると失敗し、単体テストでは順序の定義しか検証できない。
// 実スキーマへ全テーブル分のデータを投入し、実際に1件残らず消えることを確認する。
// 検証後は必ずロールバックするため、DBには何も残らない。
func TestIntegrationPurge(t *testing.T) {
	db := openTestDB(t)

	// 検証用にロールバックさせるための番兵。
	errRollback := errors.New("rollback")

	err := db.Transaction(func(tx *gorm.DB) error {
		seed(t, tx)

		// 巻き込み対象(他ユーザが対象ユーザのデッキに付けたお気に入り)を実行前に警告できること
		foreignCounts, err := countForeignAll(tx, testTargetUserId)
		require.NoError(t, err)
		require.Len(t, foreignCounts, 1)
		assert.Equal(t, "user_favorite_decks", foreignCounts[0].table)
		assert.Equal(t, int64(1), foreignCounts[0].count)

		// 全テーブルに1件以上入れてあるため、削除前の件数もすべて1件以上になる
		counts, err := countAll(tx, testTargetUserId)
		require.NoError(t, err)
		require.Len(t, counts, len(specs), "件数を数えられていないテーブルがある")

		require.Zero(t, countByQuery(t, tx, purgedQuery), "前提: 実行前は purged_at が未記録であること")

		results, err := purgeInTx(tx, testTargetUserId)
		require.NoError(t, err)

		deleted := make(map[string]int64, len(results))
		for _, r := range results {
			deleted[r.table] = r.count
		}

		for _, spec := range specs {
			assert.NotZero(t, deleted[spec.name], "%s が削除されていない", spec.name)
		}

		// purgeInTx も削除後に数え直すが、それは削除と同じ条件を使った検算でしかない。
		// ここでは条件を介さず、対象ユーザの行そのものが引けなくなったことを確かめる。

		assertTargetDataGone(t, tx)
		assertOtherUserDataKept(t, tx)

		return errRollback
	})
	require.ErrorIs(t, err, errRollback)
}

// 対象ユーザのデータが1件も残っていないことを、テーブルごとに直接数えて確認する。
func assertTargetDataGone(t *testing.T, tx *gorm.DB) {
	t.Helper()

	for _, target := range []struct {
		name  string
		query string
	}{
		{"records", "SELECT COUNT(*) FROM records WHERE user_id = '" + testTargetUserId + "'"},
		{"matches", "SELECT COUNT(*) FROM matches WHERE user_id = '" + testTargetUserId + "'"},
		{"games", "SELECT COUNT(*) FROM games WHERE user_id = '" + testTargetUserId + "'"},
		{"decks", "SELECT COUNT(*) FROM decks WHERE user_id = '" + testTargetUserId + "'"},
		{"deck_codes", "SELECT COUNT(*) FROM deck_codes WHERE user_id = '" + testTargetUserId + "'"},
		{"tags", "SELECT COUNT(*) FROM tags WHERE user_id = '" + testTargetUserId + "'"},
		{"unofficial_events", "SELECT COUNT(*) FROM unofficial_events WHERE user_id = '" + testTargetUserId + "'"},
		{"notifications", "SELECT COUNT(*) FROM notifications WHERE user_id = '" + testTargetUserId + "'"},
		// 中間テーブルは user_id を持たないため、投入したときのIDで引く
		{"deck_tags", "SELECT COUNT(*) FROM deck_tags WHERE deck_id = '" + testId("D1") + "'"},
		{"record_tags", "SELECT COUNT(*) FROM record_tags WHERE record_id = '" + testId("R1") + "'"},
		{"match_tags", "SELECT COUNT(*) FROM match_tags WHERE match_id = '" + testId("M1") + "'"},
		{"deck_code_tags", "SELECT COUNT(*) FROM deck_code_tags WHERE deck_code_id = '" + testId("C1") + "'"},
		{"match_pokemon_sprites", "SELECT COUNT(*) FROM match_pokemon_sprites WHERE match_id = '" + testId("M1") + "'"},
		{"deck_pokemon_sprites", "SELECT COUNT(*) FROM deck_pokemon_sprites WHERE deck_id = '" + testId("D1") + "'"},
	} {
		assert.Zero(t, countByQuery(t, tx, target.query), "%s に対象ユーザのデータが残っている", target.name)
	}

	// users の行は残す。usecase.User.Create の IsWithdrawn による再登録拒否を効かせ続けるため。
	assert.Equal(t, int64(1),
		countByQuery(t, tx, "SELECT COUNT(*) FROM users WHERE id = '"+testTargetUserId+"'"),
		"users の行まで消してはいけない")

	// 行を残す代わりに purged_at へ記録する。これが無いと cmd/list-deleted-users の一覧に
	// 退会ユーザとして出続け、「退会しただけ」と区別できない。
	assert.Equal(t, int64(1), countByQuery(t, tx, purgedQuery), "users.purged_at が記録されていない")
}

// purgedQuery は対象ユーザの purged_at が記録されているかを数えるSQL。
const purgedQuery = "SELECT COUNT(*) FROM users WHERE id = '" + testTargetUserId + "' AND purged_at IS NOT NULL"

// 他のユーザのデータが巻き添えで消えていないことを確認する。
func assertOtherUserDataKept(t *testing.T, tx *gorm.DB) {
	t.Helper()

	for _, target := range []struct {
		name  string
		query string
	}{
		{"decks", "SELECT COUNT(*) FROM decks WHERE user_id = '" + testOtherUserId + "'"},
		{"records", "SELECT COUNT(*) FROM records WHERE user_id = '" + testOtherUserId + "'"},
		{"matches", "SELECT COUNT(*) FROM matches WHERE user_id = '" + testOtherUserId + "'"},
		// 他ユーザ自身のデッキに付けたお気に入りは対象外
		{"user_favorite_decks", "SELECT COUNT(*) FROM user_favorite_decks WHERE user_id = '" + testOtherUserId + "' AND deck_id = '" + testId("D2") + "'"},
		// 他ユーザの対戦記録に残る対戦相手の参照(matches.opponents_user_id)は書き換えない
		{"matches.opponents_user_id", "SELECT COUNT(*) FROM matches WHERE opponents_user_id = '" + testTargetUserId + "'"},
		// 対象ユーザのデッキに付いていた他ユーザのお気に入りは、デッキごと消えるため残らない。
		// 他ユーザのお気に入りは自分のデッキぶんの1件だけになる
		{"user_favorite_decks(巻き込み後)", "SELECT COUNT(*) FROM user_favorite_decks WHERE user_id = '" + testOtherUserId + "'"},
	} {
		assert.Equal(t, int64(1), countByQuery(t, tx, target.query),
			"%s が消えている(または消すべき行が残っている)", target.name)
	}
}

// countByQuery はテスト内でSQLを直接実行して件数を数える。本体の countRows は @user_id を
// 渡す前提で、対象ユーザを介さず数えたいここでは使えないため、テスト側に用意している。
func countByQuery(t *testing.T, tx *gorm.DB, query string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, tx.Raw(query).Scan(&count).Error, query)

	return count
}

// seed は全テーブル分のテストデータを投入する。specs のテーブルすべてに1件以上入れることで、
// 「どのテーブルも実際に削除できる」ことを確認できるようにしている。
func seed(t *testing.T, tx *gorm.DB) {
	t.Helper()

	// FKの参照先になるマスタは schema.sql が投入済みのものを使う(IDは環境で変わりうるため引く)。
	// shops.id は 0 始まり(株式会社ポケモン)のため、値ではなく行が引けたかどうかで確認する。
	var badgeDefinitionId string
	requireFetched(t, tx, "SELECT id FROM badge_definitions ORDER BY id LIMIT 1", &badgeDefinitionId)

	var environmentId string
	requireFetched(t, tx, "SELECT id FROM environments ORDER BY id LIMIT 1", &environmentId)

	var shopId int
	requireFetched(t, tx, "SELECT id FROM shops ORDER BY id LIMIT 1", &shopId)

	deckId := testId("D1")
	otherDeckId := testId("D2")
	deckCodeId := testId("C1")
	recordId := testId("R1")
	otherRecordId := testId("R2")
	matchId := testId("M1")
	otherMatchId := testId("M2")
	tagId := testId("T1")
	spriteId := "test-purge-sprite"

	exec := func(query string, args ...any) {
		t.Helper()
		require.NoError(t, tx.Exec(query, args...).Error, query)
	}

	// --- ユーザ本体。対象は退会済み、比較用の他ユーザは有効なまま ---
	exec(`INSERT INTO users (id, created_at, updated_at, deleted_at, name) VALUES
	      (?, now(), now(), now(), 'テスト対象'), (?, now(), now(), NULL, 'テスト他ユーザ')`,
		testTargetUserId, testOtherUserId)

	// --- デッキとデッキコード ---
	exec(`INSERT INTO decks (id, created_at, updated_at, user_id, name) VALUES
	      (?, now(), now(), ?, 'テストデッキ'), (?, now(), now(), ?, '他ユーザのデッキ')`,
		deckId, testTargetUserId, otherDeckId, testOtherUserId)
	exec(`INSERT INTO deck_codes (id, created_at, updated_at, user_id, deck_id) VALUES (?, now(), now(), ?, ?)`,
		deckCodeId, testTargetUserId, deckId)

	// --- 記録・対戦・対局 ---
	exec(`INSERT INTO records (id, created_at, updated_at, user_id) VALUES (?, now(), now(), ?), (?, now(), now(), ?)`,
		recordId, testTargetUserId, otherRecordId, testOtherUserId)
	exec(`INSERT INTO matches
	      (id, created_at, updated_at, record_id, user_id, opponents_user_id,
	       bo3_flg, qualifying_round_flg, final_tournament_flg, victory_flg)
	      VALUES (?, now(), now(), ?, ?, NULL, false, false, false, true)`,
		matchId, recordId, testTargetUserId)
	// 他ユーザの対戦記録から、対象ユーザが対戦相手として参照されている状態を作る
	exec(`INSERT INTO matches
	      (id, created_at, updated_at, record_id, user_id, opponents_user_id,
	       bo3_flg, qualifying_round_flg, final_tournament_flg, victory_flg)
	      VALUES (?, now(), now(), ?, ?, ?, false, false, false, true)`,
		otherMatchId, otherRecordId, testOtherUserId, testTargetUserId)
	exec(`INSERT INTO games (id, created_at, updated_at, match_id, user_id) VALUES (?, now(), now(), ?, ?)`,
		testId("G1"), matchId, testTargetUserId)

	// --- タグと、各リソースへの付与(中間テーブル) ---
	exec(`INSERT INTO tags (id, created_at, updated_at, user_id, name) VALUES (?, now(), now(), ?, 'テストタグ')`,
		tagId, testTargetUserId)
	exec(`INSERT INTO deck_tags (deck_id, tag_id) VALUES (?, ?)`, deckId, tagId)
	exec(`INSERT INTO deck_code_tags (deck_code_id, tag_id) VALUES (?, ?)`, deckCodeId, tagId)
	exec(`INSERT INTO record_tags (record_id, tag_id) VALUES (?, ?)`, recordId, tagId)
	exec(`INSERT INTO match_tags (match_id, tag_id) VALUES (?, ?)`, matchId, tagId)

	// --- スプライト(マスタは無いためテスト内で作る) ---
	exec(`INSERT INTO pokemon_sprites (id, name) VALUES (?, 'テストスプライト')`, spriteId)
	exec(`INSERT INTO match_pokemon_sprites (match_id, position, pokemon_sprite_id) VALUES (?, 1, ?)`, matchId, spriteId)
	exec(`INSERT INTO deck_pokemon_sprites (deck_id, position, pokemon_sprite_id) VALUES (?, 1, ?)`, deckId, spriteId)

	// --- お気に入り。対象ユーザ自身のものと、他ユーザが対象ユーザのデッキに付けたもの(巻き込み) ---
	exec(`INSERT INTO user_favorite_decks (user_id, deck_id, created_at) VALUES (?, ?, now()), (?, ?, now()), (?, ?, now())`,
		testTargetUserId, deckId,
		testOtherUserId, deckId,
		testOtherUserId, otherDeckId)

	// --- user_id を直接持つテーブル ---
	exec(`INSERT INTO unofficial_events (id, created_at, updated_at, user_id, title, date)
	      VALUES (?, now(), now(), ?, 'テストイベント', now())`, testId("E1"), testTargetUserId)
	exec(`INSERT INTO users_players (id, created_at, updated_at, user_id, player_id)
	      VALUES (?, now(), now(), ?, '0000000000')`, testId("P1"), testTargetUserId)
	exec(`INSERT INTO user_streaks (user_id, last_recorded_week, updated_at) VALUES (?, now(), now())`, testTargetUserId)
	exec(`INSERT INTO user_daily_activities (user_id, date, category, updated_at) VALUES (?, now(), 'visit', now())`,
		testTargetUserId)
	exec(`INSERT INTO user_badges (id, created_at, user_id, badge_definition_id, achieved_at)
	      VALUES (?, now(), ?, ?, now())`, testId("B1"), testTargetUserId, badgeDefinitionId)
	exec(`INSERT INTO user_environment_badges (user_id, environment_id, achieved_at, created_at)
	      VALUES (?, ?, now(), now())`, testTargetUserId, environmentId)
	exec(`INSERT INTO notifications (id, created_at, user_id, category, title, body)
	      VALUES (?, now(), ?, 'badge', 'テスト通知', 'テスト')`, testId("N1"), testTargetUserId)
	exec(`INSERT INTO push_subscriptions (id, created_at, updated_at, user_id, endpoint, p256dh, auth)
	      VALUES (?, now(), now(), ?, 'https://example.com/test-purge', 'p256dh', 'auth')`,
		testId("S1"), testTargetUserId)
	exec(`INSERT INTO push_deliveries (id, created_at, user_id, subscription_id, campaign, status)
	      VALUES (?, now(), ?, ?, 'weekly_report', 'sent')`, testId("V1"), testTargetUserId, testId("S1"))
	exec(`INSERT INTO user_acquisitions (user_id, created_at, updated_at) VALUES (?, now(), now())`, testTargetUserId)
	exec(`INSERT INTO user_gyms (user_id, shop_id, created_at) VALUES (?, ?, now())`, testTargetUserId, shopId)
}

// specs のSQLはテーブル名・列名を手書きした文字列で、単体テストでは形しか見ていない。
// 対象データが1件も無い状態でも、実スキーマに対してSQLとして通ることを確認する。
func TestIntegrationSpecsSQL(t *testing.T) {
	db := openTestDB(t)

	errRollback := errors.New("rollback")

	for _, spec := range specs {
		t.Run("正常系_"+spec.name, func(t *testing.T) {
			_, err := countRows(db, spec.countQuery(), testTargetUserId)
			require.NoError(t, err, "件数を数えるSQLが実行できない")

			if spec.foreignWhere != "" {
				_, err := countRows(db, spec.foreignCountQuery(), testTargetUserId)
				require.NoError(t, err, "巻き込み件数を数えるSQLが実行できない")
			}

			// 削除するSQLは実際に消してしまわないよう、必ずロールバックする
			err = db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec(spec.deleteQuery(), namedUserId(testTargetUserId)).Error; err != nil {
					return err
				}

				return errRollback
			})
			require.ErrorIs(t, err, errRollback, "削除するSQLが実行できない")
		})
	}
}

// requireFetched はマスタから1件引く。値そのものではなく行が引けたかどうかで判定するため、
// 0 や空文字が正当な値であるマスタにも使える。
func requireFetched(t *testing.T, tx *gorm.DB, query string, dest any) {
	t.Helper()

	ret := tx.Raw(query).Scan(dest)
	require.NoError(t, ret.Error)
	require.Equal(t, int64(1), ret.RowsAffected, "マスタが投入されていない: %s", query)
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("VSRECORDER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VSRECORDER_TEST_DATABASE_URL が未設定のためスキップ(make integration-test で実行できます)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return db
}
