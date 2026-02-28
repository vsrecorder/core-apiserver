package authorization

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

func DeckAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		deck, err := repository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if uid != deck.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}
	}
}

func DeckGetByIdAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		deck, err := repository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if deck.PrivateFlg && uid != deck.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}
	}
}

func DeckUpdateAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckArchiveAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckUnarchiveAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckDeleteAuthorizationMiddleware(deckRepository repository.DeckInterface, recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		deck, err := deckRepository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if uid != deck.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		limit := 1
		offset := 0
		eventType := ""
		records, err := recordRepository.FindByDeckId(context.Background(), id, limit, offset, eventType)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if len(records) > 0 {
			ctx.JSON(http.StatusConflict, gin.H{"message": "cannot delete deck with records"})
			ctx.Abort()
			return
		}
	}
}
