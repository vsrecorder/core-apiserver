package validation

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestUnofficialEventCreateMiddleware(t *testing.T) {
	marshal := func(t *testing.T, req dto.UnofficialEventCreateRequest) string {
		t.Helper()
		b, err := json.Marshal(req)
		require.NoError(t, err)
		return string(b)
	}

	t.Run("正常系_イベント名と開催日が揃ったリクエストを受理して設定する", func(t *testing.T) {
		expected := dto.UnofficialEventCreateRequest{
			UnofficialEventRequest: dto.UnofficialEventRequest{
				Title: "自主大会",
				Date:  time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
			},
		}

		ctx, w := newValidationJSONContext(t, marshal(t, expected))

		UnofficialEventCreateMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expected, helper.GetUnofficialEventCreateRequest(ctx))
	})

	t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
		ctx, w := newValidationJSONContext(t, "bad data")

		UnofficialEventCreateMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_イベント名が空なら400を返す", func(t *testing.T) {
		req := dto.UnofficialEventCreateRequest{
			UnofficialEventRequest: dto.UnofficialEventRequest{
				Title: "",
				Date:  time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
			},
		}

		ctx, w := newValidationJSONContext(t, marshal(t, req))

		UnofficialEventCreateMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_開催日が未指定なら400を返す", func(t *testing.T) {
		req := dto.UnofficialEventCreateRequest{
			UnofficialEventRequest: dto.UnofficialEventRequest{
				Title: "自主大会",
			},
		}

		ctx, w := newValidationJSONContext(t, marshal(t, req))

		UnofficialEventCreateMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_イベント名が上限を超えたら400を返す", func(t *testing.T) {
		req := dto.UnofficialEventCreateRequest{
			UnofficialEventRequest: dto.UnofficialEventRequest{
				Title: strings.Repeat("あ", MaxEventTitleLength+1),
				Date:  time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
			},
		}

		ctx, w := newValidationJSONContext(t, marshal(t, req))

		UnofficialEventCreateMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
