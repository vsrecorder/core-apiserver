package infrastructure

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// setup4TonamelEventInfrastructure はhttptestサーバを立て、取得先URLを
// そのサーバへ差し替えたリポジトリを返す(外部サイトへは通信しない)。
func setup4TonamelEventInfrastructure(t *testing.T, handler http.HandlerFunc) repository.TonamelEventInterface {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := tonamelEventBaseURL
	tonamelEventBaseURL = server.URL + "/competition/"
	t.Cleanup(func() { tonamelEventBaseURL = original })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewTonamelEvent(logger)
}

func TestTonamelEventInfrastructure_FindById(t *testing.T) {
	t.Run("正常系_OGPからタイトルと説明と画像を取得する", func(t *testing.T) {
		r := setup4TonamelEventInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
			require.Equal(t, "/competition/OakZc", req.URL.Path)
			fmt.Fprint(w, `<html><head>
				<meta property="og:title" content="第23回 ACEカップ - Tonamel">
				<meta property="og:description" content="大会の説明">
				<meta property="og:image" content="https://example.com/image.png">
			</head><body></body></html>`)
		})

		ret, err := r.FindById(context.Background(), "OakZc")

		require.NoError(t, err)
		require.Equal(t, "OakZc", ret.ID)
		// タイトル末尾の " - Tonamel" は除去される
		require.Equal(t, "第23回 ACEカップ", ret.Title)
		require.Equal(t, "大会の説明", ret.Description)
		require.Equal(t, "https://example.com/image.png", ret.Image)
	})

	t.Run("正常系_og:titleが無ければtwitter:titleへフォールバックする", func(t *testing.T) {
		r := setup4TonamelEventInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, `<html><head>
				<meta name="twitter:title" content="twitterタイトル">
				<meta name="twitter:description" content="twitter説明">
			</head><body></body></html>`)
		})

		ret, err := r.FindById(context.Background(), "OakZc")

		require.NoError(t, err)
		require.Equal(t, "twitterタイトル", ret.Title)
		require.Equal(t, "twitter説明", ret.Description)
	})

	t.Run("正常系_metaタグが無ければtitleタグへフォールバックする", func(t *testing.T) {
		r := setup4TonamelEventInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, `<html><head>
				<title>  タイトルタグ - Tonamel  </title>
				<meta name="description" content="meta説明">
			</head><body></body></html>`)
		})

		ret, err := r.FindById(context.Background(), "OakZc")

		require.NoError(t, err)
		require.Equal(t, "タイトルタグ", ret.Title)
		require.Equal(t, "meta説明", ret.Description)
	})

	t.Run("異常系_404はErrRecordNotFoundを返す", func(t *testing.T) {
		r := setup4TonamelEventInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		_, err := r.FindById(context.Background(), "ERROR")

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})

	t.Run("異常系_404以外の異常ステータスはエラーを返す", func(t *testing.T) {
		r := setup4TonamelEventInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		_, err := r.FindById(context.Background(), "OakZc")

		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "503"))
	})

	t.Run("異常系_タイトルが取得できなければErrRecordNotFoundを返す", func(t *testing.T) {
		r := setup4TonamelEventInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, `<html><head></head><body></body></html>`)
		})

		_, err := r.FindById(context.Background(), "OakZc")

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})
}
