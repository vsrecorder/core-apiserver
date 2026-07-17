package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func UnofficialEventCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UnofficialEventCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// イベント名と開催日は必須
		if req.Title == "" || req.Date.IsZero() {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if exceedsLength(req.Title, MaxEventTitleLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetUnofficialEventCreateRequest(ctx, req)
	}
}
