package validation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

// デッキコードは公式サイトのURLとストレージのキーにそのまま使うため、
// 英数字とハイフン以外を含む値は作成の入口で 400 にする。
func TestDeckCodeFormatValidation(t *testing.T) {
	invalidCodes := []string{"abc/../x", "abc?x=1", "abc.png", "abc def"}

	t.Run("DeckCodeCreateMiddleware", func(t *testing.T) {
		for _, code := range invalidCodes {
			t.Run("異常系_文字種が英数字とハイフン以外なら400を返す_"+code, func(t *testing.T) {
				b, err := json.Marshal(dto.DeckCodeCreateRequest{DeckId: "01HD7Y3K8D6FDHMHTZ2GT41TD1", Code: code})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))

				DeckCodeCreateMiddleware(slog.Default())(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})

	t.Run("DeckCreateMiddleware", func(t *testing.T) {
		for _, code := range invalidCodes {
			t.Run("異常系_文字種が英数字とハイフン以外なら400を返す_"+code, func(t *testing.T) {
				b, err := json.Marshal(dto.DeckCreateRequest{Name: "test", DeckCode: code})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))

				DeckCreateMiddleware(slog.Default())(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code)
			})
		}

		t.Run("正常系_公式サイトの形式のデッキコードは受理する", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCreateRequest{Name: "test", DeckCode: "5dbFbk-uBwjqP-VVk5Vv"})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusOK, w.Code)
		})
	})
}
