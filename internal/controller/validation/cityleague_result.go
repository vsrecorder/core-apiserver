package validation

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func CityleagueResultGetByDateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		date, err := helper.ParseQueryDate(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
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
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

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

		if (fromDate.Equal(time.Time{})) != (toDate.Equal(time.Time{})) {
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

		helper.SetLeagueType(ctx, leagueType)
		helper.SetFromDate(ctx, fromDate)
		helper.SetToDate(ctx, toDate)
	}
}
