package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// isValidMatchRequest はMatchの作成/更新リクエストの整合性を検証する。
//
// 作成と更新で満たすべき整合性は同一のため、両Middlewareからこの関数を呼ぶ。
// (以前は同じ検証を各Middlewareに二重実装しており、更新側にだけ
//
//	GroupMatchVictoryFlgの検証が無い、といった乖離が生まれていた)
func isValidMatchRequest(req dto.MatchRequest) bool {
	// RecordIdが空
	if req.RecordId == "" {
		return false
	}

	// 自由入力欄が上限を超えている。memoはDB上TEXTで上限が無いため歯止めをかける
	if exceedsLength(req.Memo, MaxMemoLength) {
		return false
	}

	if exceedsLength(req.OpponentsDeckInfo, MaxOpponentsDeckInfoLength) {
		return false
	}

	gameWinningFlgs := make([]bool, 0, len(req.Games))
	for _, g := range req.Games {
		if g == nil {
			return false
		}

		if exceedsLength(g.Memo, MaxMemoLength) {
			return false
		}

		gameWinningFlgs = append(gameWinningFlgs, g.WinningFlg)
	}

	// フラグ同士・ゲーム数・勝敗整合の検証は domain 層の単一関数に集約している
	// (usecase 層でも同じ関数を使い、検証の二重実装による乖離を防ぐ)
	return entity.IsValidMatchResult(entity.MatchResultInput{
		BO3Flg:               req.BO3Flg,
		GroupMatchFlg:        req.GroupMatchFlg,
		DefaultVictoryFlg:    req.DefaultVictoryFlg,
		DefaultDefeatFlg:     req.DefaultDefeatFlg,
		VictoryFlg:           req.VictoryFlg,
		DrawFlg:              req.DrawFlg,
		GroupMatchVictoryFlg: req.GroupMatchVictoryFlg,
		GameWinningFlgs:      gameWinningFlgs,
	})
}

func MatchCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.MatchCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidMatchRequest(req.MatchRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetMatchCreateRequest(ctx, req)
	}
}

func MatchUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.MatchUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !isValidMatchRequest(req.MatchRequest) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetMatchUpdateRequest(ctx, req)
	}
}

func MatchReorderMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.MatchReorderRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// matchesが空
		if len(req.Matches) == 0 {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// idが空の要素が含まれている
		for _, m := range req.Matches {
			if m.Id == "" {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}
		}

		helper.SetMatchReorderRequest(ctx, req)
	}
}
