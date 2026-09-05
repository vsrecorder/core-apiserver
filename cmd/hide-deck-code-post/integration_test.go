package main

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// 実DBで、取り下げていない投稿だけが候補になり、非表示と解除が冪等に動くことを検証する。
// VSRECORDER_TEST_DATABASE_URL 未設定時はスキップ(make integration-test で実行される)。
func TestIntegrationHideDeckCodePost(t *testing.T) {
	dsn := os.Getenv("VSRECORDER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VSRECORDER_TEST_DATABASE_URL が未設定のためスキップ(make integration-test で実行できます)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	for _, table := range []string{"deck_code_post_imports", "deck_code_post_likes", "deck_code_posts", "deck_codes", "decks", "users"} {
		require.NoError(t, db.Exec("TRUNCATE TABLE "+table+" CASCADE").Error)
	}

	now := time.Now().Local().Truncate(time.Microsecond)
	author := "author-uid-00000000000000hide"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41HD1"
	codeId := "01HD7Y3K8D6FDHMHTZ2GT41HC1"
	activeId := "01HD7Y3K8D6FDHMHTZ2GT41HP1"
	withdrawnId := "01HD7Y3K8D6FDHMHTZ2GT41HP2"

	require.NoError(t, db.Create(model.NewUser(author, now, "投稿者", "")).Error)
	require.NoError(t, db.Create(model.NewDeck(deckId, now, sql.NullTime{}, author, "テストデッキ", true)).Error)
	require.NoError(t, db.Create(model.NewDeckCode(codeId, now, author, deckId, "kVkFF5-pQ2sZa-VFVfkV", true, "")).Error)
	// 公開中の投稿と、取り下げ済みの投稿(同じコードの過去の公開)
	withdrawnAt := now.Add(-time.Hour)
	require.NoError(t, db.Create(model.NewDeckCodePost(withdrawnId, now, now, author, deckId, codeId, now.Add(-2*time.Hour), &withdrawnAt, nil, "", "", "")).Error)
	require.NoError(t, db.Create(model.NewDeckCodePost(activeId, now, now, author, deckId, codeId, now, nil, nil, "", "", "")).Error)

	t.Run("正常系_非表示の対象は取り下げていない投稿だけで解除なら取り下げ済みも候補になる", func(t *testing.T) {
		targets, err := fetchTargets(db, withdrawnId, "", false)
		require.NoError(t, err)
		require.Empty(t, targets)

		targets, err = fetchTargets(db, "", author, false)
		require.NoError(t, err)
		require.Len(t, targets, 1)
		require.Equal(t, activeId, targets[0].ID)
		require.Equal(t, "テストデッキ", targets[0].DeckName)
		require.Nil(t, targets[0].HiddenAt)

		targets, err = fetchTargets(db, "", author, true)
		require.NoError(t, err)
		require.Len(t, targets, 2, "解除では取り下げ済みも候補")
	})

	t.Run("正常系_非表示にすると2回目は何も変わらず解除で戻る", func(t *testing.T) {
		affected, err := hide(db, []string{activeId, withdrawnId}, now)
		require.NoError(t, err)
		require.Equal(t, int64(1), affected, "取り下げ済みは更新しない")

		targets, err := fetchTargets(db, activeId, "", false)
		require.NoError(t, err)
		require.Len(t, targets, 1)
		require.NotNil(t, targets[0].HiddenAt)

		affected, err = hide(db, []string{activeId}, now.Add(time.Minute))
		require.NoError(t, err)
		require.Equal(t, int64(0), affected, "既に非表示なら更新しない")

		affected, err = unhide(db, []string{activeId}, now.Add(2*time.Minute))
		require.NoError(t, err)
		require.Equal(t, int64(1), affected)

		affected, err = unhide(db, []string{activeId}, now.Add(3*time.Minute))
		require.NoError(t, err)
		require.Equal(t, int64(0), affected, "表示中なら更新しない")

		targets, err = fetchTargets(db, activeId, "", false)
		require.NoError(t, err)
		require.Nil(t, targets[0].HiddenAt)
	})

	t.Run("正常系_非表示のまま取り下げた投稿は解除で戻せる", func(t *testing.T) {
		// 非表示 → 投稿者が取り下げ、の順で起きた状態を作る
		require.NoError(t, db.Exec("UPDATE deck_code_posts SET hidden_at = ? WHERE id = ?", now, withdrawnId).Error)

		affected, err := unhide(db, []string{withdrawnId}, now.Add(time.Minute))
		require.NoError(t, err)
		require.Equal(t, int64(1), affected, "取り下げ済みでも解除できる(公開し直しの拒否を解くため)")

		targets, err := fetchTargets(db, withdrawnId, "", true)
		require.NoError(t, err)
		require.Len(t, targets, 1)
		require.Nil(t, targets[0].HiddenAt)
		require.NotNil(t, targets[0].UnpublishedAt)
	})

	t.Run("正常系_指定が無ければ何も対象にしない", func(t *testing.T) {
		targets, err := fetchTargets(db, "", "", true)
		require.NoError(t, err)
		require.Empty(t, targets)
	})
}
