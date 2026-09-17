package validation

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func DeckGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := helper.ParseQueryLimit(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		offset, err := helper.ParseQueryOffset(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		cursor, err := helper.ParseQuerySingleCursor(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		archived, err := helper.ParseQueryArchive(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetLimit(ctx, limit)
		helper.SetOffset(ctx, offset)
		helper.SetCursor(ctx, cursor)
		helper.SetArchived(ctx, archived)
	}
}

func DeckCreateMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if req.Name == "" || exceedsLength(req.Name, MaxDeckNameLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// 長さ・文字種の確認は外部APIへの問い合わせ前に行う。
		if exceedsLength(req.DeckCode, MaxDeckCodeLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.DeckCode != "" {
			// デッキコードは公式サイトのURLとストレージのキーにそのまま使うため、文字種を限定する。
			if !entity.IsValidDeckCodeFormat(req.DeckCode) {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}

			// フロントエンド側でデッキコードの有効性を確認しているのでチェックは不要だが、念のためサーバ側でも確認したいが、
			// 大量のリクエストが来ると外部APIに負荷がかかるので、現状はコメントアウトしている。
			// もし外部APIの負荷が問題ない場合は、コメントアウトを解除してチェックを有効化する。
			//checkDeckCode(ctx, logger, req.DeckCode)
		}

		if !validatePokemonSprites(req.PokemonSprites) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !validateTagIds(req.TagIds) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDeckCreateRequest(ctx, req)
	}
}

func DeckUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if req.Name == "" || exceedsLength(req.Name, MaxDeckNameLength) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !validatePokemonSprites(req.PokemonSprites) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !validateTagIds(req.TagIds) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDeckUpdateRequest(ctx, req)
	}
}
