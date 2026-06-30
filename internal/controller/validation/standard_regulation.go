package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func StandardRegulationGetByDateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		date, err := helper.ParseQueryDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDate(ctx, date)
	}
}
