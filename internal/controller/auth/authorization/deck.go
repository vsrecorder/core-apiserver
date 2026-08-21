package authorization

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func DeckAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		deck, err := repository.FindById(ctx.Request.Context(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if uid != deck.UserId {
			apierror.ErrForbidden.JSON(ctx, err)
			return
		}
	}
}

func DeckGetByIdAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		deck, err := repository.FindById(ctx.Request.Context(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if deck.PrivateFlg && uid != deck.UserId {
			apierror.ErrForbidden.JSON(ctx, err)
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

func DeckFavoriteAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckUnfavoriteAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckDeleteAuthorizationMiddleware(deckRepository repository.DeckInterface, recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		deck, err := deckRepository.FindById(ctx.Request.Context(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if uid != deck.UserId {
			apierror.ErrForbidden.JSON(ctx, err)
			return
		}

		limit := 1
		offset := 0
		eventType := ""
		records, err := recordRepository.FindByDeckId(ctx.Request.Context(), id, limit, offset, eventType)
		if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if len(records) > 0 {
			apierror.ErrDeckHasRecords.JSON(ctx, err)
			return
		}
	}
}
