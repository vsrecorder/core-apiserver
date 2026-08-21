package authorization

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func UserAuthorizationMiddleware(repository repository.UserInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		user, err := repository.FindById(ctx.Request.Context(), id)

		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if uid != user.ID {
			apierror.ErrForbidden.JSON(ctx, err)
			return
		}
	}
}

func UserUpdateAuthorizationMiddleware(repository repository.UserInterface) gin.HandlerFunc {
	return UserAuthorizationMiddleware(repository)
}

func UserDeleteAuthorizationMiddleware(repository repository.UserInterface) gin.HandlerFunc {
	return UserAuthorizationMiddleware(repository)
}
