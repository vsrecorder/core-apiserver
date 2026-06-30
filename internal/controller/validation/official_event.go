package validation

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func OfficialEventGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		typeId, err := helper.ParseQueryTypeId(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		date, err := helper.ParseQueryDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		startDate, err := helper.ParseQueryStartDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		endDate, err := helper.ParseQueryEndDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// startDate > endDate の場合
		if !startDate.Before(endDate) && !startDate.Equal(endDate) {
			apierror.ErrBadRequest.JSON(ctx)
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
			apierror.ErrBadRequest.JSON(ctx)
			return
		} else if officialEventId <= 0 {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetOfficialEventId(ctx, uint(officialEventId))
	}
}
