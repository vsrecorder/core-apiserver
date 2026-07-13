package validation

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func CityleagueResultGetEventsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		fromDate, err := helper.ParseQueryFromDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		toDate, err := helper.ParseQueryToDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// from_date と to_date は両方指定するか、両方省略する（両方省略時は全期間が対象）
		if (fromDate.Equal(time.Time{})) != (toDate.Equal(time.Time{})) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// fromDate > toDate の場合
		if !fromDate.Before(toDate) && !fromDate.Equal(toDate) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetLeagueType(ctx, leagueType)
		helper.SetFromDate(ctx, fromDate)
		helper.SetToDate(ctx, toDate)
	}
}

func CityleagueResultGetByDateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
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

		helper.SetLeagueType(ctx, leagueType)
		helper.SetDate(ctx, date)
	}
}

func CityleagueResultGetByTermMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		fromDate, err := helper.ParseQueryFromDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		toDate, err := helper.ParseQueryToDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if (fromDate.Equal(time.Time{})) != (toDate.Equal(time.Time{})) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// fromDate > toDate の場合
		if !fromDate.Before(toDate) && !fromDate.Equal(toDate) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetLeagueType(ctx, leagueType)
		helper.SetFromDate(ctx, fromDate)
		helper.SetToDate(ctx, toDate)
	}
}

func CityleagueResultGetByOfficialEventIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		officialEventId, err := helper.ParseQueryOfficialEventId(ctx)

		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetOfficialEventId(ctx, officialEventId)
	}
}
