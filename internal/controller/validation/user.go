package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// isValidUserRequest はUserの作成/更新リクエストを検証する。
//
// 作成と更新で満たすべき条件は同一のため、両Middlewareからこの関数を呼ぶ。
func isValidUserRequest(req dto.UserRequest) bool {
	if req.Name == "" || exceedsLength(req.Name, MaxUserNameLength) {
		return false
	}

	if exceedsLength(req.ImageURL, MaxImageURLLength) {
		return false
	}

	// 空文字はisValidImageURLがスキーム無しとして弾く
	if !isValidImageURL(req.ImageURL) {
		return false
	}

	return true
}

func UserCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidUserRequest(req.UserRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetUserCreateRequest(ctx, req)
	}
}

func UserUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidUserRequest(req.UserRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetUserUpdateRequest(ctx, req)
	}
}
