package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// TonamelEventGetByIdMiddleware は GET /tonamel_events/:id の大会IDの形式を検証する。
//
// このIDはそのまま tonamel.com のURLに連結して取得に行くため、形式を見ずに通すと
// 任意のパスのページを取得させられる(取得先ホストは固定なので、影響は tonamel.com 内の
// 別ページの OGP を返す範囲に留まるが、外部サイトへの無駄な問い合わせにもなる)。
func TonamelEventGetByIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !entity.IsValidTonamelEventId(helper.GetId(ctx)) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}
	}
}
