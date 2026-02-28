package authorization

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

func RecordAuthorizationMiddleware(repository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		record, err := repository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if uid != record.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}
	}
}

func RecordGetByIdAuthorizationMiddleware(repository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		record, err := repository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		// 非公開レコードである場合、ユーザIDが一致しないと403を返す
		if record.PrivateFlg && uid != record.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
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
