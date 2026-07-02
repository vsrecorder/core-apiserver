package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func OpponentDeckUsageStatGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		yearMonth, err := helper.ParseQueryYearMonth(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}
		helper.SetYearMonth(ctx, yearMonth)

		environmentId := helper.GetQueryEnvironmentId(ctx)
		helper.SetEnvironmentId(ctx, environmentId)

		season, err := helper.ParseQuerySeason(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}
		helper.SetSeason(ctx, season)

		regulationId := helper.GetQueryRegulationId(ctx)
		helper.SetRegulationId(ctx, regulationId)

		helper.SetDeckId(ctx, helper.GetQueryDeckId(ctx))
	}
}
