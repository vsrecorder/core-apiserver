package authorization

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

func DeckCodeAuthorizationMiddleware(repository repository.DeckCodeInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		deckcode, err := repository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if uid != deckcode.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}
	}
}

func DeckCodeUpdateAuthorizationMiddleware(repository repository.DeckCodeInterface) gin.HandlerFunc {
	return DeckCodeAuthorizationMiddleware(repository)
}

func DeckCodeDeleteAuthorizationMiddleware(deckcodeRepository repository.DeckCodeInterface, recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		deckcode, err := deckcodeRepository.FindById(context.Background(), id)
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if uid != deckcode.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		limit := 1
		offset := 0
		records, err := recordRepository.FindByDeckCodeId(context.Background(), id, limit, offset)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if len(records) > 0 {
			ctx.JSON(http.StatusConflict, gin.H{"message": "cannot delete deckcode with records"})
			ctx.Abort()
			return
		}
	}
}
