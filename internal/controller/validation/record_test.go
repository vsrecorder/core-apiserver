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

func TestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"RecordGetMiddleware":    test_RecordGetMiddleware,
		"RecordCreateMiddleware": test_RecordCreateMiddleware,
		"RecordUpdateMiddleware": test_RecordUpdateMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_RecordGetMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		expectedLimit := 10
		expectedOffset := 0

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedLimit, helper.GetLimit(ginContext))
		require.Equal(t, expectedOffset, helper.GetOffset(ginContext))
	})

	t.Run("正常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		expectedLimit := 20
		expectedOffset := 10

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?limit=%d&offset=%d", expectedLimit, expectedOffset), nil)
		require.NoError(t, err)

		ginContext.Request = req

		helper.SetLimit(ginContext, expectedLimit)
		helper.SetOffset(ginContext, expectedOffset)

		middleware := RecordGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedLimit, helper.GetLimit(ginContext))
		require.Equal(t, expectedOffset, helper.GetOffset(ginContext))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/?limit=a&offset=0", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetMiddleware()
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

		middleware := RecordGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_RecordCreateMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		privateFlg := false

		expected := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetRecordCreateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		tonamelEventId := "61ozP"
		privateFlg := false

		expected := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: 0,
				TonamelEventId:  tonamelEventId,
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetRecordCreateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		expected := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: 0,
				TonamelEventId:  "",
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetRecordCreateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader("bad data"))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		tonamelEventId := "61ozP"
		friendId := ""
		privateFlg := false

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		tonamelEventId := ""
		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#04", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(0)
		tonamelEventId := "61ozP"
		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#05", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		tonamelEventId := "61ozP"
		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#06", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(0)
		tonamelEventId := ""
		friendId := ""
		privateFlg := false

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_RecordUpdateMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		privateFlg := false

		expected := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		actual := helper.GetRecordUpdateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		tonamelEventId := "61ozP"
		privateFlg := false

		expected := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: 0,
				TonamelEventId:  tonamelEventId,
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		actual := helper.GetRecordUpdateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		expected := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: 0,
				TonamelEventId:  "",
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		actual := helper.GetRecordUpdateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader("bad data"))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		tonamelEventId := "61ozP"
		friendId := ""
		privateFlg := false

		data := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		tonamelEventId := ""
		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		data := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#04", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(0)
		tonamelEventId := "61ozP"
		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		data := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#05", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(10000)
		tonamelEventId := "61ozP"
		friendId := "d4385mX98abtmLny3qxlmBlBLIu1"
		privateFlg := false

		data := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#06", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		officialEventId := uint(0)
		tonamelEventId := ""
		friendId := ""
		privateFlg := false

		data := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  tonamelEventId,
				FriendId:        friendId,
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
