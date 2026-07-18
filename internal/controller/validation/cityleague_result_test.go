package validation

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestCityleagueResultValidation(t *testing.T) {
	t.Run("CityleagueResultGetEventsMiddleware", func(t *testing.T) {
		t.Run("正常系_期間未指定なら全期間として通過する", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "league_type=4")

			CityleagueResultGetEventsMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uint(4), helper.GetLeagueType(ctx))
			require.True(t, helper.GetFromDate(ctx).IsZero())
			require.True(t, helper.GetToDate(ctx).IsZero())
		})

		t.Run("正常系_期間指定をコンテキストに設定する", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "from_date=2026-07-01&to_date=2026-07-31")

			CityleagueResultGetEventsMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local), helper.GetFromDate(ctx))
			require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), helper.GetToDate(ctx))
		})

		t.Run("異常系_league_typeが不正なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "league_type=abc")

			CityleagueResultGetEventsMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_from_dateのみ指定なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "from_date=2026-07-01")

			CityleagueResultGetEventsMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_from_dateがto_dateより後なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "from_date=2026-07-31&to_date=2026-07-01")

			CityleagueResultGetEventsMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("CityleagueResultGetByDateMiddleware", func(t *testing.T) {
		t.Run("正常系_league_typeとdateをコンテキストに設定する", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "league_type=4&date=2026-07-18")

			CityleagueResultGetByDateMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uint(4), helper.GetLeagueType(ctx))
			require.Equal(t, time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local), helper.GetDate(ctx))
		})

		t.Run("異常系_dateの形式が不正なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "date=20260718")

			CityleagueResultGetByDateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("CityleagueResultGetByTermMiddleware", func(t *testing.T) {
		t.Run("正常系_期間指定をコンテキストに設定する", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "from_date=2026-07-01&to_date=2026-07-31")

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local), helper.GetFromDate(ctx))
			require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), helper.GetToDate(ctx))
		})

		t.Run("異常系_to_dateのみ指定なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "to_date=2026-07-31")

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_from_dateがto_dateより後なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "from_date=2026-07-31&to_date=2026-07-01")

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("CityleagueResultGetByOfficialEventIdMiddleware", func(t *testing.T) {
		t.Run("正常系_official_event_idをコンテキストに設定する", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "official_event_id=606466")

			CityleagueResultGetByOfficialEventIdMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uint(606466), helper.GetOfficialEventId(ctx))
		})

		t.Run("異常系_official_event_idが数値でなければ400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "official_event_id=abc")

			CityleagueResultGetByOfficialEventIdMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
