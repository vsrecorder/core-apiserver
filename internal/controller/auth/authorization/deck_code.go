package authorization

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func DeckCodeAuthorizationMiddleware(repository repository.DeckCodeInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		deckcode, err := repository.FindById(context.Background(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if uid != deckcode.UserId {
			apierror.ErrForbidden.JSON(ctx)
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
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		deckcode, err := deckcodeRepository.FindById(context.Background(), id)
		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if uid != deckcode.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		limit := 1
		offset := 0
		records, err := recordRepository.FindByDeckCodeId(context.Background(), id, limit, offset)
		if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if len(records) > 0 {
			apierror.ErrDeckCodeHasRecords.JSON(ctx)
			return
		}
	}
}
