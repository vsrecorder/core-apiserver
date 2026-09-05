package authorization

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// DeckCodePostDeleteAuthorizationMiddleware は投稿の取り下げを投稿者本人に限る。
// 「デッキの公開中の投稿一覧」(GET /decks/:id/deck_code_posts)の認可は、デッキの所有者
// 確認そのものなので既存の DeckAuthorizationMiddleware を使う。
func DeckCodePostDeleteAuthorizationMiddleware(repository repository.DeckCodePostInterface) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)
		uid := helper.GetUID(ctx)

		if uid == "" {
			apierror.ErrForbidden.JSON(ctx)
			return
		}

		post, err := repository.FindLiteById(ctx.Request.Context(), id)
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		} else if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}

		if uid != post.UserId {
			apierror.ErrForbidden.JSON(ctx)
			return
		}
	}
}
