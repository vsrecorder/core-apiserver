package validation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newValidationGETContext は指定したクエリ文字列を持つGETリクエストのgin.Contextを返す。
func newValidationGETContext(t *testing.T, rawQuery string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	url := "/"
	if rawQuery != "" {
		url = "/?" + rawQuery
	}

	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	ctx.Request = req

	return ctx, w
}

// newValidationJSONContext は指定したボディを持つPOSTリクエストのgin.Contextを返す。
func newValidationJSONContext(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	require.NoError(t, err)

	ctx.Request = req

	return ctx, w
}

func TestIsValidImageURL(t *testing.T) {
	t.Run("正常系_ホストを持つhttpsのURLを受け入れる", func(t *testing.T) {
		require.True(t, isValidImageURL("https://example.com/image.png"))
	})

	t.Run("異常系_httpスキームは拒否する", func(t *testing.T) {
		require.False(t, isValidImageURL("http://example.com/image.png"))
	})

	t.Run("異常系_javascriptスキームは大文字混じりでも拒否する", func(t *testing.T) {
		require.False(t, isValidImageURL("javascript:alert(1)"))
		require.False(t, isValidImageURL("JavaScript:alert(1)"))
	})

	t.Run("異常系_dataスキームは拒否する", func(t *testing.T) {
		require.False(t, isValidImageURL("data:text/html;base64,PHNjcmlwdD4="))
	})

	t.Run("異常系_ホストを持たない値は拒否する", func(t *testing.T) {
		require.False(t, isValidImageURL("https:/image.png"))
	})
}

func TestIsValidTCGMeisterURL(t *testing.T) {
	t.Run("正常系_任意項目のため空文字は受け入れる", func(t *testing.T) {
		require.True(t, isValidTCGMeisterURL(""))
	})

	t.Run("正常系_httpとhttpsの両方を受け入れる", func(t *testing.T) {
		require.True(t, isValidTCGMeisterURL("https://tcg.sfc-jpn.jp/tour.asp?tid=123456"))
		require.True(t, isValidTCGMeisterURL("http://tcg.sfc-jpn.jp/tour.asp?tid=123456"))
	})

	t.Run("異常系_javascriptスキームは大文字混じりでも拒否する", func(t *testing.T) {
		require.False(t, isValidTCGMeisterURL("javascript:alert(1)"))
		require.False(t, isValidTCGMeisterURL("JavaScript:alert(1)"))
	})

	t.Run("異常系_ホストを持たない値は拒否する", func(t *testing.T) {
		require.False(t, isValidTCGMeisterURL("https:/tour.asp"))
	})
}

func TestExceedsLength(t *testing.T) {
	t.Run("正常系_上限以内の文字列はfalseを返す", func(t *testing.T) {
		require.False(t, exceedsLength("あいう", 3))
	})

	t.Run("正常系_バイト数ではなく文字数で判定する", func(t *testing.T) {
		// "あいう"は9バイトだが3文字なので上限3文字に収まる
		require.False(t, exceedsLength("あいう", 3))
		require.True(t, exceedsLength("あいうえ", 3))
	})

	t.Run("正常系_上限を超えた文字列はtrueを返す", func(t *testing.T) {
		require.True(t, exceedsLength("abcd", 3))
	})
}
