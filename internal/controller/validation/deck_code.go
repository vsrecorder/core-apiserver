package validation

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func DeckCodeCreateMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCodeCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.Code == "" || exceedsLength(req.Code, MaxDeckCodeLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if exceedsLength(req.Memo, MaxMemoLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// フロントエンド側でデッキコードの有効性を確認しているのでチェックは不要だが、念のためサーバ側でも確認したいが、
		// 大量のリクエストが来ると外部APIに負荷がかかるので、現状はコメントアウトしている。
		// もし外部APIの負荷が問題ない場合は、コメントアウトを解除してチェックを有効化する。
		//checkDeckCode(ctx, logger, req.Code)

		helper.SetDeckCodeCreateRequest(ctx, req)
	}
}

func DeckCodeUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCodeUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if exceedsLength(req.Memo, MaxMemoLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDeckCodeUpdateRequest(ctx, req)
	}
}
