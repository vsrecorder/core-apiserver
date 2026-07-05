package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func WeeklyDeckUsageStatGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		week, err := helper.ParseQueryWeek(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}
		helper.SetWeek(ctx, week)
	}
}
