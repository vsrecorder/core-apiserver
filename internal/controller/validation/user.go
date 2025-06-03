package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func UserCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.Name == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.ImageURL == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetUserCreateRequest(ctx, req)
	}
}

func UserUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.Name == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.ImageURL == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetUserUpdateRequest(ctx, req)
	}
}
