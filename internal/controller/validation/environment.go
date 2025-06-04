package validation

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func EnvironmentGetByDateMiddleware() gin.HandlerFunc {
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

func EnvironmentGetByTermMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fromDate, err := helper.ParseQueryFromDate(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		toDate, err := helper.ParseQueryToDate(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if (fromDate == time.Time{}) != (toDate == time.Time{}) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// fromDate > toDate の場合
		if !fromDate.Before(toDate) && !fromDate.Equal(toDate) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetFromDate(ctx, fromDate)
		helper.SetToDate(ctx, toDate)
	}
}
