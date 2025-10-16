package validation

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func OfficialEventGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		typeId, err := helper.ParseQueryTypeId(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

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

		startDate, err := helper.ParseQueryStartDate(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		endDate, err := helper.ParseQueryEndDate(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// startDate > endDate の場合
		if !startDate.Before(endDate) && !startDate.Equal(endDate) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetTypeId(ctx, typeId)
		helper.SetLeagueType(ctx, leagueType)
		helper.SetDate(ctx, date)
		helper.SetStartDate(ctx, startDate)
		helper.SetEndDate(ctx, endDate)
	}
}

func OfficialEventGetByIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)

		officialEventId, err := strconv.Atoi(id)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		} else if officialEventId <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetOfficialEventId(ctx, uint(officialEventId))
	}
}
