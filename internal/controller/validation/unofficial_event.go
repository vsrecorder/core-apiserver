package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func UnofficialEventCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UnofficialEventCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// イベント名と開催日は必須
		if req.Title == "" || req.Date.IsZero() {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetUnofficialEventCreateRequest(ctx, req)
	}
}
