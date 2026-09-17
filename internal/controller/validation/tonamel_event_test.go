package validation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

// Tonamel の大会IDはそのまま外部サイトのURLに連結するため、形式外の値は入口で 400 にする。
func TestTonamelEventIdValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContextWithId := func(t *testing.T, id string) (*gin.Context, *httptest.ResponseRecorder) {
		t.Helper()

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		ctx.Request = req
		ctx.Params = gin.Params{{Key: "id", Value: id}}

		return ctx, w
	}

	t.Run("TonamelEventGetByIdMiddleware", func(t *testing.T) {
		t.Run("正常系_英数字のIDは通す", func(t *testing.T) {
			ctx, w := newContextWithId(t, "61ozP")

			TonamelEventGetByIdMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.False(t, ctx.IsAborted())
		})

		for _, id := range []string{"..", "a/b", "ab?c", "abcdefghi"} {
			t.Run("異常系_形式外のIDは400を返す_"+id, func(t *testing.T) {
				ctx, w := newContextWithId(t, id)

				TonamelEventGetByIdMiddleware()(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code)
				require.True(t, ctx.IsAborted())
			})
		}
	})

	t.Run("RecordCreateMiddleware", func(t *testing.T) {
		t.Run("異常系_Tonamelの大会IDが形式外なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					EventDate:      testValidationEventDate,
					TonamelEventId: "../x",
				},
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			RecordCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("RecordUpdateMiddleware", func(t *testing.T) {
		t.Run("異常系_Tonamelの大会IDが形式外なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.RecordUpdateRequest{
				RecordRequest: dto.RecordRequest{
					EventDate:      testValidationEventDate,
					TonamelEventId: "abcdefghi",
				},
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			RecordUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
