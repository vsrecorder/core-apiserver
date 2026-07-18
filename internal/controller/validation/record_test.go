package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestRecordValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"RecordGetMiddleware":                 test_RecordGetMiddleware,
		"RecordCreateMiddleware":              test_RecordCreateMiddleware,
		"RecordUpdateMiddleware":              test_RecordUpdateMiddleware,
		"RecordUpdateMiddlewareTCGMeisterURL": test_RecordUpdateMiddlewareTCGMeisterURL,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_RecordGetMiddleware(t *testing.T) {
	t.Run("正常系_クエリ未指定ならデフォルトのlimitとoffsetを設定して通過する", func(t *testing.T) {
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

	t.Run("正常系_limitとoffset指定を受け付ける", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		expectedLimit := 20
		expectedOffset := 10

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?limit=%d&offset=%d", expectedLimit, expectedOffset), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedLimit, helper.GetLimit(ginContext))
		require.Equal(t, expectedOffset, helper.GetOffset(ginContext))
	})

	t.Run("異常系_limitが数値でなければ400を返す", func(t *testing.T) {
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

	t.Run("異常系_offsetが数値でなければ400を返す", func(t *testing.T) {
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
	t.Run("正常系_公式イベント指定のリクエストを受理する", func(t *testing.T) {
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

	t.Run("正常系_Tonamelイベント指定のリクエストを受理する", func(t *testing.T) {
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

	t.Run("正常系_フレンド対戦指定のリクエストを受理する", func(t *testing.T) {
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

	t.Run("正常系_自由形式", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		eventDate := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
		privateFlg := true

		expected := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId:   0,
				TonamelEventId:    "",
				FriendId:          "",
				DeckId:            "",
				PrivateFlg:        privateFlg,
				TCGMeisterURL:     "",
				Memo:              "",
				EventDate:         eventDate,
				UnofficialEventId: "01HD7Y3K8D6FDHMHTZ2GT41TN2",
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

	t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
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

	t.Run("異常系_公式イベントとTonamelイベントの併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_公式イベントとフレンド対戦の併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_Tonamelイベントとフレンド対戦の併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_3種のイベントすべての併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_イベント未指定は400を返す", func(t *testing.T) {
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

	t.Run("異常系_自由形式と公式イベントの併用", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// 公式イベントと自由形式イベント名が同時に指定されている場合は bad request
		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId:   10000,
				TonamelEventId:    "",
				FriendId:          "",
				DeckId:            "",
				PrivateFlg:        true,
				TCGMeisterURL:     "",
				Memo:              "",
				EventDate:         time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC),
				UnofficialEventId: "01HD7Y3K8D6FDHMHTZ2GT41TN2",
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

	// webapp は tcg_meister_url を <a href> にそのまま入れて描画するため、
	// javascript: のような危険なスキームは保存させない。
	// webapp側の入力チェックはAPIを直接叩けば素通りするので、ここが防御線になる。
	for _, tcgMeisterURL := range []string{
		"javascript:alert(1)",
		// url.Parseがスキームを小文字化するため、大文字混じりも同じく弾ける
		"JavaScript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==",
		// ホストを持たない値
		"https:/tour.asp",
	} {
		t.Run("異常系_危険なTCGマイスターURL_"+tcgMeisterURL, func(t *testing.T) {
			w := httptest.NewRecorder()
			ginContext, _ := gin.CreateTestContext(w)

			data := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: uint(10000),
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      false,
					TCGMeisterURL:   tcgMeisterURL,
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

	// 正規のTCGマイスターURLは従来どおり通す。
	// webappの入力欄がhttp/httpsの両方を受け付けているため、どちらも許容する。
	for _, tcgMeisterURL := range []string{
		"https://tcg.sfc-jpn.jp/tour.asp?tid=123456",
		"http://tcg.sfc-jpn.jp/tour.asp?tid=123456",
	} {
		t.Run("正常系_正規のTCGマイスターURL_"+tcgMeisterURL, func(t *testing.T) {
			w := httptest.NewRecorder()
			ginContext, _ := gin.CreateTestContext(w)

			expected := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: uint(10000),
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      false,
					TCGMeisterURL:   tcgMeisterURL,
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

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetRecordCreateRequest(ginContext))
		})
	}
}

func test_RecordUpdateMiddleware(t *testing.T) {
	t.Run("正常系_公式イベント指定のリクエストを受理する", func(t *testing.T) {
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

	t.Run("正常系_Tonamelイベント指定のリクエストを受理する", func(t *testing.T) {
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

	t.Run("正常系_フレンド対戦指定のリクエストを受理する", func(t *testing.T) {
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

	t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
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

	t.Run("異常系_公式イベントとTonamelイベントの併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_公式イベントとフレンド対戦の併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_Tonamelイベントとフレンド対戦の併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_3種のイベントすべての併用は400を返す", func(t *testing.T) {
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

	t.Run("異常系_イベント未指定は400を返す", func(t *testing.T) {
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

// 更新経路にも同じ検証が効いていることを担保する。
// 危険なURLは作成で弾いても、更新で入れられれば同じことになる。
func test_RecordUpdateMiddlewareTCGMeisterURL(t *testing.T) {
	t.Run("異常系_TCGマイスターURLにjavascriptスキーム", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		data := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: uint(10000),
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      false,
				TCGMeisterURL:   "javascript:alert(1)",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("正常系_正規のTCGマイスターURL", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		expected := dto.RecordUpdateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: uint(10000),
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      false,
				TCGMeisterURL:   "https://tcg.sfc-jpn.jp/tour.asp?tid=123456",
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

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expected, helper.GetRecordUpdateRequest(ginContext))
	})
}
