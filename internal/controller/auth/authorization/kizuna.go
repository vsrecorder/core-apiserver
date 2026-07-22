package authorization

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// きずなLv.は本人の記録の積み重ねそのものなので、本人以外には見せない。
func KizunaAuthorizationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		if uid != id {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}
