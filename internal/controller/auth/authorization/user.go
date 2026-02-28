package authorization

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

func UserAuthorizationMiddleware(repository repository.UserInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		user, err := repository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if uid != user.ID {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
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
