package validation

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// cityleagueResultsRangeMaxDays はシティリーグ結果(入賞者つき)を1度に引ける期間の最大日数。
//
// 期間は呼び出し側が指定し、ログインなしで呼べる。cityleague_results は入賞者1人につき1行で
// 12万行を超えており、上限が無いと全期間を1回指定されるだけで、その全行にプレイヤー名を結合して
// 返すことになる(応答は約15MB)。作業機で測ると、その1回でプロセスのメモリが 36MB から 117MB に
// 増え、4本重ねると 455MB になった(2026-11-26)。本番のコンテナは 128MB が上限
// (docker-compose.yml の mem_limit)なので、1〜2回で落とせる。
//
// 正規の利用で最も長いのは、webapp の結果一覧が1回に遡る14日ぶん(CITYLEAGUE_SEARCH_DAYS)。
// 過去2週間の本番のログでも最長は4日だった。約2倍の余裕を取る。31日ぶんでも応答は 0.6MB ほど。
//
// 入賞者を含まない /events は全期間でも 0.7MB で、sitemap とハブが全期間で使うので絞らない。
const cityleagueResultsRangeMaxDays = 31

func CityleagueResultGetEventsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		fromDate, err := helper.ParseQueryFromDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		toDate, err := helper.ParseQueryToDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		// from_date と to_date は両方指定するか、両方省略する（両方省略時は全期間が対象）
		if (fromDate.Equal(time.Time{})) != (toDate.Equal(time.Time{})) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// fromDate > toDate の場合
		if !fromDate.Before(toDate) && !fromDate.Equal(toDate) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetLeagueType(ctx, leagueType)
		helper.SetFromDate(ctx, fromDate)
		helper.SetToDate(ctx, toDate)
	}
}

func CityleagueResultGetByDateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		date, err := helper.ParseQueryDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetLeagueType(ctx, leagueType)
		helper.SetDate(ctx, date)
	}
}

func CityleagueResultGetByTermMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		fromDate, err := helper.ParseQueryFromDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		toDate, err := helper.ParseQueryToDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if (fromDate.Equal(time.Time{})) != (toDate.Equal(time.Time{})) {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		// fromDate > toDate の場合
		if !fromDate.Before(toDate) && !fromDate.Equal(toDate) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if toDate.Sub(fromDate) > cityleagueResultsRangeMaxDays*24*time.Hour {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetLeagueType(ctx, leagueType)
		helper.SetFromDate(ctx, fromDate)
		helper.SetToDate(ctx, toDate)
	}
}

func CityleagueResultGetByOfficialEventIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		officialEventId, err := helper.ParseQueryOfficialEventId(ctx)

		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetOfficialEventId(ctx, officialEventId)
	}
}
