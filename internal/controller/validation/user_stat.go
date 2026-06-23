package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func UserStatGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		yearMonth, err := helper.ParseQueryYearMonth(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}
		helper.SetYearMonth(ctx, yearMonth)

		environmentId := helper.GetQueryEnvironmentId(ctx)
		helper.SetEnvironmentId(ctx, environmentId)

		season, err := helper.ParseQuerySeason(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}
		helper.SetSeason(ctx, season)
	}
}
