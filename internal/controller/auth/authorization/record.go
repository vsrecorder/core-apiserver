package authorization

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func RecordAuthorizationMiddleware(repository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		record, err := repository.FindById(context.Background(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if uid != record.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func RecordGetByIdAuthorizationMiddleware(repository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		record, err := repository.FindById(context.Background(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		// 非公開レコードである場合、ユーザIDが一致しないと403を返す
		if record.PrivateFlg && uid != record.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func RecordUpdateAuthorizationMiddleware(repository repository.RecordInterface) gin.HandlerFunc {
	return RecordAuthorizationMiddleware(repository)
}

func RecordDeleteAuthorizationMiddleware(repository repository.RecordInterface) gin.HandlerFunc {
	return RecordAuthorizationMiddleware(repository)
}
