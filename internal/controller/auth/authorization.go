package auth

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

func DeckUpdateAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckArchiveAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckUnarchiveAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func DeckDeleteAuthorizationMiddleware(repository repository.DeckInterface) gin.HandlerFunc {
	return DeckAuthorizationMiddleware(repository)
}

func MatchAuthorizationMiddleware(repository repository.MatchInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}

		match, err := repository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if uid != match.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}
	}
}

func MatchGetByIdAuthorizationMiddleware(matchRepository repository.MatchInterface, recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		match, err := matchRepository.FindById(context.Background(), id)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		record, err := recordRepository.FindById(context.Background(), match.RecordId)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if record.PrivateFlg && uid != record.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
			return
		}
	}
}

func MatchGetByRecordIdAuthorizationMiddleware(recordRepository repository.RecordInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		recordId := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		record, err := recordRepository.FindById(context.Background(), recordId)

		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			ctx.Abort()
			return
		}

		if record.PrivateFlg && uid != record.UserId {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			ctx.Abort()
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
