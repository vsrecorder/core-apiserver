package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func UserStatHistoryGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		period := helper.GetQueryPeriod(ctx)
		switch period {
		case "3months", "6months", "season":
		case "":
			period = "3months"
		default:
			apierror.ErrBadRequest.JSON(ctx)
			return
		}
		helper.SetPeriod(ctx, period)

		season, err := helper.ParseQuerySeason(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}
		helper.SetSeason(ctx, season)
	}
}
