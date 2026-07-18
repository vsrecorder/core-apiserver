package authorization

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// 「パスパラメータのidと認証済みuidが一致する本人のみ許可する」同型の
// ミドルウェア群をまとめてテストする。
func TestSelfOnlyAuthorizationMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middlewares := map[string]gin.HandlerFunc{
		"CalendarAuthorizationMiddleware":              CalendarAuthorizationMiddleware(),
		"DeckUsageStatAuthorizationMiddleware":         DeckUsageStatAuthorizationMiddleware(),
		"OldestRecordAuthorizationMiddleware":          OldestRecordAuthorizationMiddleware(),
		"OpponentDeckUsageStatAuthorizationMiddleware": OpponentDeckUsageStatAuthorizationMiddleware(),
	}

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	newContext := func(t *testing.T, id string, uid string) (*gin.Context, *httptest.ResponseRecorder) {
		t.Helper()

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		if uid != "" {
			helper.SetUID(ctx, uid)
		}

		ctx.Params = append(ctx.Params, gin.Param{Key: "id", Value: id})

		return ctx, w
	}

	for name, middleware := range middlewares {
		t.Run(name, func(t *testing.T) {
			t.Run("正常系_本人なら通過する", func(t *testing.T) {
				ctx, w := newContext(t, uid, uid)

				middleware(ctx)

				require.Equal(t, http.StatusOK, w.Code)
			})

			t.Run("異常系_未認証なら403を返す", func(t *testing.T) {
				ctx, w := newContext(t, uid, "")

				middleware(ctx)

				require.Equal(t, http.StatusForbidden, w.Code)
			})

			t.Run("異常系_他人のIDなら403を返す", func(t *testing.T) {
				ctx, w := newContext(t, "KBp7roRDZobZg1t0OPzFR1kvLeO2", uid)

				middleware(ctx)

				require.Equal(t, http.StatusForbidden, w.Code)
			})
		})
	}
}
