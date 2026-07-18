package validation

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestDeckUsageStatGetMiddleware(t *testing.T) {
	t.Run("正常系_指定した集計条件をコンテキストに設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "all_time=true&year_month=2026-07&environment_id=sv11&season=2026&regulation_id=regulation-g")

		DeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.True(t, helper.GetAllTime(ctx))
		require.Equal(t, "2026-07", helper.GetYearMonth(ctx))
		require.Equal(t, "sv11", helper.GetEnvironmentId(ctx))
		require.Equal(t, "2026", helper.GetSeason(ctx))
		require.Equal(t, "regulation-g", helper.GetRegulationId(ctx))
	})

	t.Run("正常系_未指定ならデフォルト値を設定して通過する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "")

		DeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.False(t, helper.GetAllTime(ctx))
		require.Empty(t, helper.GetYearMonth(ctx))
	})

	t.Run("異常系_all_timeが真偽値でなければ400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "all_time=abc")

		DeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_year_monthの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "year_month=202607")

		DeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_seasonの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "season=abc")

		DeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOpponentDeckUsageStatGetMiddleware(t *testing.T) {
	t.Run("正常系_指定した集計条件とdeck_idをコンテキストに設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "year_month=2026-07&environment_id=sv11&season=2026&regulation_id=regulation-g&deck_id=01HD7Y3K8D6FDHMHTZ2GT41TN2")

		OpponentDeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "2026-07", helper.GetYearMonth(ctx))
		require.Equal(t, "sv11", helper.GetEnvironmentId(ctx))
		require.Equal(t, "2026", helper.GetSeason(ctx))
		require.Equal(t, "regulation-g", helper.GetRegulationId(ctx))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", helper.GetDeckId(ctx))
	})

	t.Run("異常系_year_monthの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "year_month=abc")

		OpponentDeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_seasonの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "season=abc")

		OpponentDeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserStatGetMiddleware(t *testing.T) {
	t.Run("正常系_指定した集計条件をコンテキストに設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "year_month=2026-07&environment_id=sv11&season=2026&regulation_id=regulation-g")

		UserStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "2026-07", helper.GetYearMonth(ctx))
		require.Equal(t, "sv11", helper.GetEnvironmentId(ctx))
		require.Equal(t, "2026", helper.GetSeason(ctx))
		require.Equal(t, "regulation-g", helper.GetRegulationId(ctx))
	})

	t.Run("異常系_year_monthの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "year_month=abc")

		UserStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_seasonの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "season=abc")

		UserStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserStatHistoryGetMiddleware(t *testing.T) {
	t.Run("正常系_定義済みのperiodはそのまま設定する", func(t *testing.T) {
		for _, period := range []string{"3months", "6months", "season"} {
			ctx, w := newValidationGETContext(t, "period="+period)

			UserStatHistoryGetMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, period, helper.GetPeriod(ctx))
		}
	})

	t.Run("正常系_period未指定なら3monthsを設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "")

		UserStatHistoryGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "3months", helper.GetPeriod(ctx))
	})

	t.Run("正常系_deck_idをコンテキストに設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "deck_id=01HD7Y3K8D6FDHMHTZ2GT41TN2")

		UserStatHistoryGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", helper.GetDeckId(ctx))
	})

	t.Run("異常系_未定義のperiodなら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "period=1year")

		UserStatHistoryGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_seasonの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "season=abc")

		UserStatHistoryGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserStatRecentGetMiddleware(t *testing.T) {
	t.Run("正常系_定義済みのcountはlimitとして設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "count=50")

		UserStatRecentGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 50, helper.GetLimit(ctx))
	})

	t.Run("正常系_count未指定なら20を設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "")

		UserStatRecentGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 20, helper.GetLimit(ctx))
	})

	t.Run("異常系_未定義のcountなら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "count=25")

		UserStatRecentGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestWeeklyDeckUsageStatGetMiddleware(t *testing.T) {
	t.Run("正常系_week指定をコンテキストに設定する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "week=2026-07-13")

		WeeklyDeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "2026-07-13", helper.GetWeek(ctx))
	})

	t.Run("正常系_week未指定なら空文字を設定して通過する", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "")

		WeeklyDeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, helper.GetWeek(ctx))
	})

	t.Run("異常系_weekの形式が不正なら400を返す", func(t *testing.T) {
		ctx, w := newValidationGETContext(t, "week=2026/07/13")

		WeeklyDeckUsageStatGetMiddleware()(ctx)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
