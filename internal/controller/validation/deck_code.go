package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func DeckCodeCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCodeCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.Code == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		} else {
			checkDeckCode(ctx, req.Code)
		}

		helper.SetDeckCodeCreateRequest(ctx, req)
	}
}

func DeckCodeUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCodeUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetDeckCodeUpdateRequest(ctx, req)
	}
}
