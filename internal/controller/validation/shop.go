package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func ShopGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		keyword, err := helper.ParseQueryKeyword(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		// キーワードが無いと店舗マスタの先頭 limit 件が返るだけで意味がなく、
		// 「検索したのに探している店が出ない」と読めてしまうため弾く。
		if keyword == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// 既定件数と上限は usecase.Shop が持つ。ここは形式だけを見て 0(未指定)を通す。
		limit, err := helper.ParseQueryLimitOptional(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetKeyword(ctx, keyword)
		helper.SetLimit(ctx, limit)
	}
}
