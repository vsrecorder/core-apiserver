package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func DeckUsageStatGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		allTime, err := helper.ParseQueryAllTime(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}
		helper.SetAllTime(ctx, allTime)

		yearMonth, err := helper.ParseQueryYearMonth(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}
		helper.SetYearMonth(ctx, yearMonth)

		// 週(月曜始まり)での絞り込み。週内の任意日 YYYY-MM-DD を受け、月曜への正規化は usecase 側が行う
		week, err := helper.ParseQueryWeek(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}
		helper.SetWeek(ctx, week)

		environmentId := helper.GetQueryEnvironmentId(ctx)
		helper.SetEnvironmentId(ctx, environmentId)

		season, err := helper.ParseQuerySeason(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}
		helper.SetSeason(ctx, season)

		// 期間の絞り込みに使うスタンダードレギュレーション(マーク期間)
		standardRegulationId := helper.GetQueryStandardRegulationId(ctx)
		helper.SetStandardRegulationId(ctx, standardRegulationId)

		// レギュレーション区分(スタンダード/エクストラ/殿堂)での絞り込み
		helper.SetRegulationId(ctx, helper.ParseQueryRegulationId(ctx))
	}
}
