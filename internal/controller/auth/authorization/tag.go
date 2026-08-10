package authorization

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// TagAuthorizationMiddleware は対象のタグが認証ユーザー自身のものかを確認する。
// 他人のタグのリネーム・削除を防ぐ。deck の同種ミドルウェアを踏襲。
func TagAuthorizationMiddleware(repository repository.TagInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		tag, err := repository.FindById(context.Background(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		if uid != tag.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}

func TagUpdateAuthorizationMiddleware(repository repository.TagInterface) gin.HandlerFunc {
	return TagAuthorizationMiddleware(repository)
}

func TagDeleteAuthorizationMiddleware(repository repository.TagInterface) gin.HandlerFunc {
	return TagAuthorizationMiddleware(repository)
}
