package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func OldestRecordGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		helper.SetDeckId(ctx, helper.GetQueryDeckId(ctx))
	}
}
