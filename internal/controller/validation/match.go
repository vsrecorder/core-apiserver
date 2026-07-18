package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
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

	for _, g := range req.Games {
		if g == nil {
			return false
		}

		if exceedsLength(g.Memo, MaxMemoLength) {
			return false
		}
	}

	// DefaultVictoryFlgとDefaultDefeatFlgの両方がtrue
	if req.DefaultVictoryFlg && req.DefaultDefeatFlg {
		return false
	}

	// DefaultVictoryFlgがtrueなのにVictoryFlgがfalse
	if req.DefaultVictoryFlg && !req.VictoryFlg {
		return false
	}

	// DefaultDefeatFlgがtrueなのにVictoryFlgがtrue
	if req.DefaultDefeatFlg && req.VictoryFlg {
		return false
	}

	// 不戦勝/不戦敗は対戦が行われていないため、Gameが存在してはならない
	isDefault := req.DefaultVictoryFlg || req.DefaultDefeatFlg
	if isDefault && len(req.Games) > 0 {
		return false
	}

	// チームの勝敗(GroupMatchVictoryFlg)を持てるのはチーム戦のBO1のみ
	if req.GroupMatchVictoryFlg && (req.BO3Flg || !req.GroupMatchFlg) {
		return false
	}

	// 不戦勝/不戦敗の場合はGameが存在しないため、ここから先の検証は行わない
	if isDefault {
		return true
	}

	if req.BO3Flg {
		// BO3(2本先取)は2ゲーム(2-0)または3ゲーム(2-1)で決着する
		if len(req.Games) != 2 && len(req.Games) != 3 {
			return false
		}

		if len(req.Games) == 2 {
			// 2ゲームで決着した場合は2連勝(2-0)か2連敗(0-2)であり、
			// どちらのゲームの勝敗も対戦全体の勝敗と一致する
			//
			// | victory | game 1 | game 2 |
			// ------------------------------
			// |  true   |  true  |  true  |
			// |  false  |  false |  false |
			//
			if req.Games[0].WinningFlg != req.VictoryFlg || req.Games[1].WinningFlg != req.VictoryFlg {
				return false
			}
		}

		if len(req.Games) == 3 {
			// 3ゲーム目が行われるのは1勝1敗で並んだ場合のみ。
			// 1・2ゲーム目が同じ勝敗(2-0 or 0-2)なら既に決着しているため不正
			if req.Games[0].WinningFlg == req.Games[1].WinningFlg {
				return false
			}

			// 1勝1敗で並んだ場合、3ゲーム目の勝敗が対戦全体の勝敗になる
			//
			// | victory | game 1 | game 2 | game 3 |
			// --------------------------------------
			// |  true   |  true  |  false |  true  |
			// |  true   |  false |  true  |  true  |
			// |  false  |  true  |  false |  false |
			// |  false  |  false |  true  |  false |
			//
			if req.Games[2].WinningFlg != req.VictoryFlg {
				return false
			}
		}
	} else {
		// BO1(1本勝負)はちょうど1ゲームで決着する
		if len(req.Games) != 1 {
			return false
		}

		// GameのWinningFlgとMatchのVictoryFlgが異なる
		if req.Games[0].WinningFlg != req.VictoryFlg {
			return false
		}
	}

	return true
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
