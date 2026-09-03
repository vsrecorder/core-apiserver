package validation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestChampionsleagueResultGetByChampionsleagueScheduleIdMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(t *testing.T, query string) (*httptest.ResponseRecorder, *gin.Context) {
		t.Helper()

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/championsleague_results?"+query, nil)

		ChampionsleagueResultGetByChampionsleagueScheduleIdMiddleware()(ctx)

		return w, ctx
	}

	t.Run("正常系_大会IDをコンテキストへ格納する", func(t *testing.T) {
		_, ctx := run(t, "championsleague_schedule_id=cl2027_yokohama")

		require.False(t, ctx.IsAborted())
		require.Equal(t, "cl2027_yokohama", helper.GetChampionsleagueScheduleId(ctx))
	})

	t.Run("異常系_大会IDが無い場合は400を返す", func(t *testing.T) {
		w, ctx := run(t, "")

		require.True(t, ctx.IsAborted())
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_大会IDが桁数を超える場合は400を返す", func(t *testing.T) {
		w, ctx := run(t, "championsleague_schedule_id="+strings.Repeat("a", 64))

		require.True(t, ctx.IsAborted())
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
