package validation

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func EnvironmentGetByDateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		date, err := helper.ParseQueryDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDate(ctx, date)
	}
}

func EnvironmentGetByTermMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
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

		helper.SetFromDate(ctx, fromDate)
		helper.SetToDate(ctx, toDate)
	}
}
