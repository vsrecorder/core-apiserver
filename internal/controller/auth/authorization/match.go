package authorization

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func MatchAuthorizationMiddleware(repository repository.MatchInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		match, err := repository.FindById(context.Background(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if uid != match.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func MatchGetByIdAuthorizationMiddleware(matchRepository repository.MatchInterface, recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		match, err := matchRepository.FindById(context.Background(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		record, err := recordRepository.FindById(context.Background(), match.RecordId)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if record.PrivateFlg && uid != record.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func MatchGetByRecordIdAuthorizationMiddleware(recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		recordId := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		record, err := recordRepository.FindById(context.Background(), recordId)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if record.PrivateFlg && uid != record.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func MatchUpdateAuthorizationMiddleware(repository repository.MatchInterface) gin.HandlerFunc {
	return MatchAuthorizationMiddleware(repository)
}

func MatchDeleteAuthorizationMiddleware(repository repository.MatchInterface) gin.HandlerFunc {
	return MatchAuthorizationMiddleware(repository)
}
