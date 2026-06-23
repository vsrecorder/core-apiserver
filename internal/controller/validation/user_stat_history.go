package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}
		helper.SetPeriod(ctx, period)

		season, err := helper.ParseQuerySeason(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}
		helper.SetSeason(ctx, season)
	}
}
