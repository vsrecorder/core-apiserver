package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func MatchCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.MatchCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// RecordIdが空
		if req.RecordId == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultVictoryFlgとDefaultDefeatFlgの両方がtrue
		if req.DefaultVictoryFlg && req.DefaultDefeatFlg {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultVictoryFlgがtrueなのにVictoryFlgがfalse
		if req.DefaultVictoryFlg && !req.VictoryFlg {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultDefeatFlgがtrueなのにVictoryFlgがtrue
		if req.DefaultDefeatFlg && req.VictoryFlg {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultVictoryFlg or DefaultDefeatFlgがtrueなのにGamesが存在している
		if req.DefaultVictoryFlg || req.DefaultDefeatFlg {
			if len(req.Games) > 0 {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}
		}

		if ((!req.BO3Flg && !req.GroupMatchVictoryFlg) || req.BO3Flg || !req.GroupMatchFlg) && req.GroupMatchVictoryFlg {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.BO3Flg {
			// BO3なのに試合数が1つ or 3つを超えている
			if len(req.Games) == 1 || len(req.Games) > 3 {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}

			if len(req.Games) == 2 {
				// 試合数が2つの場合はそれぞれのGameのWinningFlgとMatchのVictoryFlgが同じであるべき
				if !((req.Games[0].WinningFlg == req.VictoryFlg) && (req.Games[1].WinningFlg == req.VictoryFlg)) {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}
			}

			if len(req.Games) == 3 {
				// 1試合目も2試合目も勝っているのに試合数が3つある
				//
				// | victory | game 1 | game 2 | game 3 |
				// --------------------------------------
				// |  true   |  true  |  true  |  true  |
				// |  true   |  true  |  true  |  false |
				// |  false  |  true  |  true  |  true  |
				// |  false  |  true  |  true  |  false |
				//
				if req.Games[0].WinningFlg && req.Games[1].WinningFlg {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}

				// 1試合目も2試合目も負けているのに試合数が3つある
				//
				// | victory | game 1 | game 2 | game 3 |
				// --------------------------------------
				// |  true   |  false |  false |  true  |
				// |  true   |  false |  false |  false |
				// |  false  |  false |  false |  true  |
				// |  false  |  false |  false |  false |
				//
				if !req.Games[0].WinningFlg && !req.Games[1].WinningFlg {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}

				//
				// | victory | game 1 | game 2 | game 3 |   XOR  |
				// -----------------------------------------------
				// |  true   |  true  |  false |  true  |  true  |
				// |  true   |  false |  true  |  true  |  true  |
				// |  false  |  true  |  false |  false |  true  |
				// |  false  |  false |  true  |  false |  true  |
				//
				// |  false  |  true  |  false |  true  |  false |
				// |  false  |  false |  true  |  true  |  false |
				// |  true   |  true  |  false |  false |  false |
				// |  true   |  false |  true  |  false |  false |
				//
				if !(req.Games[0].WinningFlg != req.Games[1].WinningFlg != req.Games[2].WinningFlg != req.VictoryFlg) {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}
			}
		} else {
			// BO1なのに試合数が1つを超えている
			if len(req.Games) > 1 {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}

			if !req.DefaultVictoryFlg && !req.DefaultDefeatFlg {
				// GameのWinningFlgとMatchのVictoryFlgが異なる
				if req.Games[0].WinningFlg != req.VictoryFlg {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}
			}
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

		// RecordIdが空
		if req.RecordId == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultVictoryFlgとDefaultDefeatFlgの両方がtrue
		if req.DefaultVictoryFlg && req.DefaultDefeatFlg {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultVictoryFlgがtrueなのにVictoryFlgがfalse
		if req.DefaultVictoryFlg && !req.VictoryFlg {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultDefeatFlgがtrueなのにVictoryFlgがtrue
		if req.DefaultDefeatFlg && req.VictoryFlg {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// DefaultVictoryFlg or DefaultDefeatFlgがtrueなのにGamesが存在している
		if req.DefaultVictoryFlg || req.DefaultDefeatFlg {
			if len(req.Games) > 0 {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}
		}

		if req.BO3Flg {
			// BO3なのに試合数が1つ or 3つを超えている
			if len(req.Games) == 1 || len(req.Games) > 3 {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}

			if len(req.Games) == 2 {
				// 試合数が2つの場合はそれぞれのGameのWinningFlgとMatchのVictoryFlgが同じであるべき
				if !((req.Games[0].WinningFlg == req.VictoryFlg) && (req.Games[1].WinningFlg == req.VictoryFlg)) {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}
			}

			if len(req.Games) == 3 {
				// 1試合目も2試合目も勝っているのに試合数が3つある
				//
				// | victory | game 1 | game 2 | game 3 |
				// --------------------------------------
				// |  true   |  true  |  true  |  true  |
				// |  true   |  true  |  true  |  false |
				// |  false  |  true  |  true  |  true  |
				// |  false  |  true  |  true  |  false |
				//
				if req.Games[0].WinningFlg && req.Games[1].WinningFlg {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}

				// 1試合目も2試合目も負けているのに試合数が3つある
				//
				// | victory | game 1 | game 2 | game 3 |
				// --------------------------------------
				// |  true   |  false |  false |  true  |
				// |  true   |  false |  false |  false |
				// |  false  |  false |  false |  true  |
				// |  false  |  false |  false |  false |
				//
				if !req.Games[0].WinningFlg && !req.Games[1].WinningFlg {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}

				//
				// | victory | game 1 | game 2 | game 3 |   XOR  |
				// -----------------------------------------------
				// |  true   |  true  |  false |  true  |  true  |
				// |  true   |  false |  true  |  true  |  true  |
				// |  false  |  true  |  false |  false |  true  |
				// |  false  |  false |  true  |  false |  true  |
				//
				// |  false  |  true  |  false |  true  |  false |
				// |  false  |  false |  true  |  true  |  false |
				// |  true   |  true  |  false |  false |  false |
				// |  true   |  false |  true  |  false |  false |
				//
				if !(req.Games[0].WinningFlg != req.Games[1].WinningFlg != req.Games[2].WinningFlg != req.VictoryFlg) {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}
			}
		} else {
			// BO1なのに試合数が1つを超えている
			if len(req.Games) > 1 {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}

			if !req.DefaultVictoryFlg && !req.DefaultDefeatFlg {
				// GameのWinningFlgとMatchのVictoryFlgが異なる
				if req.Games[0].WinningFlg != req.VictoryFlg {
					apierror.ErrBadRequest.JSON(ctx)
					return
				}
			}
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
