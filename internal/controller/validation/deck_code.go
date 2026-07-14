package validation

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func DeckCodeCreateMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCodeCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.Code == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		} else {
			checkDeckCode(ctx, logger, req.Code)
		}

		helper.SetDeckCodeCreateRequest(ctx, req)
	}
}

func DeckCodeUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCodeUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDeckCodeUpdateRequest(ctx, req)
	}
}
