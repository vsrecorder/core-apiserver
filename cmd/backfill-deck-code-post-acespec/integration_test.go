package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// 実DBと、deckcard-api / webapp を模したサーバで、空の ACE SPEC が埋まり
// OGP 画像の作り直し(個別ページの取得)が走ることを検証する。
// VSRECORDER_TEST_DATABASE_URL 未設定時はスキップ(make integration-test で実行される)。
func TestIntegrationBackfillAceSpec(t *testing.T) {
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
	author := "author-uid-000000000000backfl"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41BF1"
	// 1件目: ACE SPEC が入っているデッキ(埋まる) / 2件目: 入っていないデッキ(空のまま)
	codeWithAce := "kVkFF5-pQ2sZa-VFVfkV"
	codeWithoutAce := "gLnLnn-BGH9q3-9LnQLg"
	codeIdWithAce := "01HD7Y3K8D6FDHMHTZ2GT41BC1"
	codeIdWithoutAce := "01HD7Y3K8D6FDHMHTZ2GT41BC2"
	postWithAce := "01HD7Y3K8D6FDHMHTZ2GT41BP1"
	postWithoutAce := "01HD7Y3K8D6FDHMHTZ2GT41BP2"

	require.NoError(t, db.Create(model.NewUser(author, now, "投稿者", "")).Error)
	require.NoError(t, db.Create(model.NewDeck(deckId, now, sql.NullTime{}, author, "テストデッキ", true)).Error)
	require.NoError(t, db.Create(model.NewDeckCode(codeIdWithAce, now, author, deckId, codeWithAce, true, "")).Error)
	require.NoError(t, db.Create(model.NewDeckCode(codeIdWithoutAce, now, author, deckId, codeWithoutAce, true, "")).Error)
	require.NoError(t, db.Create(model.NewDeckCodePost(postWithAce, now, now, author, deckId, codeIdWithAce, now, nil, nil, "", "", "")).Error)
	require.NoError(t, db.Create(model.NewDeckCodePost(postWithoutAce, now, now, author, deckId, codeIdWithoutAce, now.Add(-time.Minute), nil, nil, "", "", "")).Error)

	// deckcard-api: ACE SPEC を返すコードと、204(入っていない)を返すコード
	deckCardServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1beta/deckcards/"+codeWithAce+"/acespec" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"card_id":47870,"card_name":"アンフェアスタンプ","image_url":"https://example.com/47870.jpg"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer deckCardServer.Close()

	// webapp: 個別ページの取得(＝OGP 画像の生成)
	var refreshed atomic.Int64
	var refreshedPath atomic.Value
	webappServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshed.Add(1)
		refreshedPath.Store(r.URL.Path)
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer webappServer.Close()

	deckCard := infrastructure.NewDeckCard(deckCardServer.URL)
	client := webappServer.Client()
	ctx := context.Background()

	t.Run("正常系_確認だけの実行では何も変えない", func(t *testing.T) {
		ret, err := run(ctx, db, deckCard, client, webappServer.URL, options{dryRun: true, refreshOgp: true})

		require.NoError(t, err)
		require.Equal(t, 1, ret.updated)
		require.Equal(t, 1, ret.noAceSpec)
		require.Equal(t, 0, ret.failed)
		require.Equal(t, int64(0), refreshed.Load(), "確認だけの実行では個別ページを取得しない")

		var stored string
		require.NoError(t, db.Raw("SELECT ace_spec_card_id FROM deck_code_posts WHERE id = ?", postWithAce).Scan(&stored).Error)
		require.Empty(t, stored)
	})

	t.Run("正常系_埋めてOGP画像を作り直し2回目は何も起きない", func(t *testing.T) {
		ret, err := run(ctx, db, deckCard, client, webappServer.URL, options{dryRun: false, refreshOgp: true})

		require.NoError(t, err)
		require.Equal(t, 1, ret.updated)
		require.Equal(t, 1, ret.noAceSpec, "ACE SPEC が入っていないデッキは書き込まない")
		require.Equal(t, 0, ret.failed)
		require.Equal(t, int64(1), refreshed.Load())
		require.Equal(t, "/shared_decks/"+postWithAce, refreshedPath.Load())

		var post model.DeckCodePost
		require.NoError(t, db.Where("id = ?", postWithAce).First(&post).Error)
		require.Equal(t, "47870", post.AceSpecCardId)
		require.Equal(t, "アンフェアスタンプ", post.AceSpecCardName)
		require.Equal(t, "https://example.com/47870.jpg", post.AceSpecImageURL)

		// 2回目: 埋まった投稿は対象から外れ、書き込みも画像の作り直しも起きない
		ret, err = run(ctx, db, deckCard, client, webappServer.URL, options{dryRun: false, refreshOgp: true})
		require.NoError(t, err)
		require.Equal(t, 0, ret.updated)
		require.Equal(t, 1, ret.noAceSpec)
		require.Equal(t, int64(1), refreshed.Load(), "作り直しは増えない")
	})

	t.Run("正常系_取り下げた投稿は対象にしない", func(t *testing.T) {
		require.NoError(t, db.Exec("UPDATE deck_code_posts SET ace_spec_card_id = '', unpublished_at = ? WHERE id = ?", now, postWithAce).Error)

		targets, err := fetchTargets(db, options{})
		require.NoError(t, err)
		for _, target := range targets {
			require.NotEqual(t, postWithAce, target.ID)
		}
	})

	t.Run("正常系_ユーザと件数で絞れる", func(t *testing.T) {
		targets, err := fetchTargets(db, options{userId: "someone-else"})
		require.NoError(t, err)
		require.Empty(t, targets)

		targets, err = fetchTargets(db, options{userId: author, limit: 1})
		require.NoError(t, err)
		require.Len(t, targets, 1)
	})

	t.Run("異常系_deckcard-apiの失敗は件数に数えて次へ進む", func(t *testing.T) {
		failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failing.Close()

		ret, err := run(ctx, db, infrastructure.NewDeckCard(failing.URL), client, webappServer.URL, options{dryRun: false, refreshOgp: true})

		require.NoError(t, err)
		require.Equal(t, 0, ret.updated)
		require.Equal(t, 1, ret.failed)
	})
}

// entity 側の型を使っていないことによる未使用 import を避けるための参照。
var _ = entity.AceSpecCard{}
