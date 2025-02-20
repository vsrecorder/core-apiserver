package validation

import (
	"encoding/json"
	"fmt"
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
	t.Run("正常系_#01", func(t *testing.T) {
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

	t.Run("正常系_#02", func(t *testing.T) {
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

	t.Run("正常系_#03", func(t *testing.T) {
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

	t.Run("異常系_#01", func(t *testing.T) {
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

	t.Run("異常系_#02", func(t *testing.T) {
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

	t.Run("異常系_#02", func(t *testing.T) {
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
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"

		expected := dto.DeckCreateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           name,
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetDeckCreateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader("bad data"))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""

		expected := dto.DeckCreateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           name,
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_DeckUpdateMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"

		expected := dto.DeckUpdateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           name,
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckUpdateMiddleware()
		middleware(ginContext)

		actual := helper.GetDeckUpdateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader("bad data"))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""

		expected := dto.DeckCreateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           name,
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := DeckUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
