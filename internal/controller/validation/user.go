package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func UserCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.Name == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.ImageURL == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetUserCreateRequest(ctx, req)
	}
}

func UserUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.Name == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.ImageURL == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetUserUpdateRequest(ctx, req)
	}
}
