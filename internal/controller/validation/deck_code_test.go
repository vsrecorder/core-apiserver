package validation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestDeckCodeValidation(t *testing.T) {
	t.Run("DeckCodeCreateMiddleware", func(t *testing.T) {
		// 正常系はデッキコードの実在確認で公式サイトへ問い合わせるため、
		// ここでは外部通信の前に弾かれる異常系のみを扱う。
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
