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

	// 期間幅の上限。入賞者つきの結果は、全期間を1回指定されるだけで本番のコンテナのメモリ上限に届く
	t.Run("CityleagueResultGetByTermMiddleware_期間幅の上限", func(t *testing.T) {
		const layout = "2006-01-02"
		fromDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
		query := func(toDate time.Time) string {
			return "from_date=" + fromDate.Format(layout) + "&to_date=" + toDate.Format(layout)
		}

		t.Run("正常系_上限ちょうどの期間指定を受け付ける", func(t *testing.T) {
			toDate := fromDate.AddDate(0, 0, cityleagueResultsRangeMaxDays)
			ctx, w := newValidationGETContext(t, query(toDate))

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.False(t, ctx.IsAborted())
			require.Equal(t, fromDate, helper.GetFromDate(ctx))
			require.Equal(t, toDate, helper.GetToDate(ctx))
		})

		// webapp の結果一覧が1回に遡る範囲(14日ぶん)は収まる
		t.Run("正常系_結果一覧が1回に遡る14日ぶんを受け付ける", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, query(fromDate.AddDate(0, 0, 13)))

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.False(t, ctx.IsAborted())
		})

		t.Run("異常系_上限を超える期間指定なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, query(fromDate.AddDate(0, 0, cityleagueResultsRangeMaxDays+1)))

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.True(t, ctx.IsAborted())
		})

		// 全期間を1回で引こうとする指定(この上限を入れた理由そのもの)
		t.Run("異常系_全期間の指定なら400を返す", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "from_date=2015-01-01&to_date=2030-01-01")

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.True(t, ctx.IsAborted())
		})

		// 期間を指定しない要求(開催日・大会IDで引く要求)は、この検証を素通りする
		t.Run("正常系_期間未指定なら通過する", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "official_event_id=606466")

			CityleagueResultGetByTermMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.False(t, ctx.IsAborted())
		})

		// 入賞者を含まない /events は全期間で使うので、絞らない
		t.Run("正常系_events_は上限を超える期間でも受け付ける", func(t *testing.T) {
			ctx, w := newValidationGETContext(t, "from_date=2015-01-01&to_date=2030-01-01")

			CityleagueResultGetEventsMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.False(t, ctx.IsAborted())
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
