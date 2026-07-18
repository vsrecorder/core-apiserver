package validation

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// dateクエリのみを検証する同型のミドルウェア群をまとめてテストする。
func TestDateQueryMiddlewares(t *testing.T) {
	middlewares := map[string]gin.HandlerFunc{
		"ChampionshipSeriesGetByDateMiddleware": ChampionshipSeriesGetByDateMiddleware(),
		"CityleagueScheduleGetByDateMiddleware": CityleagueScheduleGetByDateMiddleware(),
		"StandardRegulationGetByDateMiddleware": StandardRegulationGetByDateMiddleware(),
	}

	for name, middleware := range middlewares {
		t.Run(name, func(t *testing.T) {
			t.Run("正常系_date未指定ならゼロ値の日付を設定して通過する", func(t *testing.T) {
				ctx, w := newValidationGETContext(t, "")

				middleware(ctx)

				require.Equal(t, http.StatusOK, w.Code)
				require.True(t, helper.GetDate(ctx).IsZero())
			})

			t.Run("正常系_date指定時はローカル時刻の日付として設定する", func(t *testing.T) {
				ctx, w := newValidationGETContext(t, "date=2026-07-18")

				middleware(ctx)

				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local), helper.GetDate(ctx))
			})

			t.Run("異常系_日付形式が不正なら400を返す", func(t *testing.T) {
				ctx, w := newValidationGETContext(t, "date=20260718")

				middleware(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code)
			})
		})
	}
}
