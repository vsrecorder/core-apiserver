package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func DeckGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := helper.ParseQueryLimit(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		offset, err := helper.ParseQueryOffset(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		cursor, err := helper.ParseQuerySingleCursor(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		archived, err := helper.ParseQueryArchive(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetLimit(ctx, limit)
		helper.SetOffset(ctx, offset)
		helper.SetCursor(ctx, cursor)
		helper.SetArchived(ctx, archived)
	}
}

func DeckCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.Name == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.DeckCode != "" {
			checkDeckCode(ctx, req.DeckCode)
		}

		helper.SetDeckCreateRequest(ctx, req)
	}
}

func DeckUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.Name == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDeckUpdateRequest(ctx, req)
	}
}
