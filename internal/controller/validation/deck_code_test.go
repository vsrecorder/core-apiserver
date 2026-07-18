package validation

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestDeckCodeValidation(t *testing.T) {
	// overrideDeckIDCheckAPI はデッキコードの実在確認APIをhttptestサーバへ差し替える
	// (公式サイトへは通信しない)。
	overrideDeckIDCheckAPI := func(t *testing.T, handler http.HandlerFunc) {
		t.Helper()

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		original := DeckIDCheckURL
		DeckIDCheckURL = server.URL
		t.Cleanup(func() { DeckIDCheckURL = original })
	}

	t.Run("DeckCodeCreateMiddleware", func(t *testing.T) {
		t.Run("正常系_実在するデッキコードのリクエストを受理して設定する", func(t *testing.T) {
			overrideDeckIDCheckAPI(t, func(w http.ResponseWriter, req *http.Request) {
				require.NoError(t, req.ParseForm())
				require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", req.PostForm.Get("deckID"))
				fmt.Fprint(w, `{"result":1,"deckID":"5dbFbk-uBwjqP-VVk5Vv","existence":1}`)
			})

			expected := dto.DeckCodeCreateRequest{
				DeckId: "01HD7Y3K8D6FDHMHTZ2GT41TD1",
				Code:   "5dbFbk-uBwjqP-VVk5Vv",
			}

			b, err := json.Marshal(expected)
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetDeckCodeCreateRequest(ctx))
		})

		t.Run("異常系_実在しないデッキコードなら400を返す", func(t *testing.T) {
			overrideDeckIDCheckAPI(t, func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, `{"result":0,"errMsg":"deck not found","existence":0}`)
			})

			b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: "XXXXXX-YYYYYY-ZZZZZZ"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_確認先がメンテナンス中なら503を返す", func(t *testing.T) {
			overrideDeckIDCheckAPI(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			})

			b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: "5dbFbk-uBwjqP-VVk5Vv"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusServiceUnavailable, w.Code)
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_コードが空なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: ""})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_コードが上限を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeCreateRequest{Code: strings.Repeat("x", MaxDeckCodeLength+1)})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_メモが上限を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeCreateRequest{
				Code: "5dbFbk-uBwjqP-VVk5Vv",
				Memo: strings.Repeat("あ", MaxMemoLength+1),
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("DeckCodeUpdateMiddleware", func(t *testing.T) {
		t.Run("正常系_更新リクエストを受理してコンテキストに設定する", func(t *testing.T) {
			expected := dto.DeckCodeUpdateRequest{PrivateCodeFlg: true, Memo: "メモ"}

			b, err := json.Marshal(expected)
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, expected, helper.GetDeckCodeUpdateRequest(ctx))
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			ctx, w := newValidationJSONContext(t, "bad data")

			DeckCodeUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_メモが上限を超えたら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCodeUpdateRequest{Memo: strings.Repeat("あ", MaxMemoLength+1)})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCodeUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
