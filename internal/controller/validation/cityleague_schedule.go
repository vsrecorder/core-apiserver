package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func CityleagueScheduleGetByDateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		date, err := helper.ParseQueryDate(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetDate(ctx, date)
	}
}
