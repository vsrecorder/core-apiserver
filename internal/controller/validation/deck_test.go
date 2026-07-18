package validation

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestDeckValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"DeckGetMiddleware":    test_DeckGetMiddleware,
		"DeckCreateMiddleware": test_DeckCreateMiddleware,
		"DeckUpdateMiddleware": test_DeckUpdateMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_DeckGetMiddleware(t *testing.T) {
	t.Run("正常系_クエリ未指定ならデフォルト値を設定して通過する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		expectedLimit := 10
		expectedOffset := 0
		expectedArchived := false

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedLimit, helper.GetLimit(ginContext))
		require.Equal(t, expectedOffset, helper.GetOffset(ginContext))
		require.Equal(t, expectedArchived, helper.GetArchived(ginContext))
	})

	t.Run("正常系_limitとoffsetとarchived指定を受け付ける", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		expectedLimit := 20
		expectedOffset := 10
		expectedArchived := true

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?limit=%d&offset=%d&archived=%t", expectedLimit, expectedOffset, expectedArchived), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedLimit, helper.GetLimit(ginContext))
		require.Equal(t, expectedOffset, helper.GetOffset(ginContext))
		require.Equal(t, expectedArchived, helper.GetArchived(ginContext))
	})

	t.Run("正常系_archivedがfalse指定でも受け付ける", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		expectedLimit := 20
		expectedOffset := 10
		expectedArchived := false

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?limit=%d&offset=%d&archived=%t", expectedLimit, expectedOffset, expectedArchived), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedLimit, helper.GetLimit(ginContext))
		require.Equal(t, expectedOffset, helper.GetOffset(ginContext))
		require.Equal(t, expectedArchived, helper.GetArchived(ginContext))
	})

	t.Run("異常系_limitが数値でなければ400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/?limit=a&offset=0", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_offsetが数値でなければ400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/?limit=10&offset=a", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_archivedが真偽値でなければ400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/?limit=10&offset=0&archived=a", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_DeckCreateMiddleware(t *testing.T) {
	t.Run("正常系_デッキコード付きリクエストを受理してコンテキストに設定する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"

		expected := dto.DeckCreateRequest{
			Name:               name,
			PrivateFlg:         false,
			DeckCode:           "48Yx8x-cJUK50-xxcxKJ",
			PrivateDeckCodeFlg: false,
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware(slog.Default())
		middleware(ginContext)

		actual := helper.GetDeckCreateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("正常系_デッキコードなしのリクエストも受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"

		expected := dto.DeckCreateRequest{
			Name:               name,
			PrivateFlg:         false,
			DeckCode:           "",
			PrivateDeckCodeFlg: false,
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware(slog.Default())
		middleware(ginContext)

		actual := helper.GetDeckCreateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader("bad data"))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware(slog.Default())
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_nameが空なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""

		expected := dto.DeckCreateRequest{
			Name:               name,
			PrivateFlg:         false,
			DeckCode:           "",
			PrivateDeckCodeFlg: false,
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware(slog.Default())
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_デッキコードの形式が不正なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"

		expected := dto.DeckCreateRequest{
			Name:               name,
			PrivateFlg:         false,
			DeckCode:           "XXXXXX-YYYYYY-ZZZZZZ",
			PrivateDeckCodeFlg: false,
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware(slog.Default())
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

}

func test_DeckUpdateMiddleware(t *testing.T) {
	t.Run("正常系_更新リクエストを受理してコンテキストに設定する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"

		expected := dto.DeckUpdateRequest{
			Name:       name,
			PrivateFlg: false,
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckUpdateMiddleware()
		middleware(ginContext)

		actual := helper.GetDeckUpdateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader("bad data"))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_nameが空なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""

		expected := dto.DeckUpdateRequest{
			Name:       name,
			PrivateFlg: false,
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

}
