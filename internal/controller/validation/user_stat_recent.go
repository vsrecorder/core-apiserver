package validation

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func UserStatRecentGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		countStr := helper.GetQueryCount(ctx)
		var count int
		switch countStr {
		case "20", "30", "40", "50", "100":
			count, _ = strconv.Atoi(countStr)
		case "":
			count = 20
		default:
			apierror.ErrBadRequest.JSON(ctx)
			return
		}
		helper.SetLimit(ctx, count)

		helper.SetDeckId(ctx, helper.GetQueryDeckId(ctx))
	}
}
