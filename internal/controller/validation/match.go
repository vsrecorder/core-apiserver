package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func MatchCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.MatchCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// RecordIdが空
		if req.RecordId == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultVictoryFlgとDefaultDefeatFlgの両方がtrue
		if req.DefaultVictoryFlg && req.DefaultDefeatFlg {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultVictoryFlgがtrueなのにVictoryFlgがfalse
		if req.DefaultVictoryFlg && !req.VictoryFlg {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultDefeatFlgがtrueなのにVictoryFlgがtrue
		if req.DefaultDefeatFlg && req.VictoryFlg {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultVictoryFlg or DefaultDefeatFlgがtrueなのにGamesが存在している
		if req.DefaultVictoryFlg || req.DefaultDefeatFlg {
			if len(req.Games) > 0 {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		}

		if req.BO3Flg {
			// BO3なのに試合数が1つ or 3つを超えている
			if len(req.Games) == 1 || len(req.Games) > 3 {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}

			if len(req.Games) == 2 {
				// 試合数が2つの場合はそれぞれのGameのWinningFlgとMatchのVictoryFlgが同じであるべき
				if !((req.Games[0].WinningFlg == req.VictoryFlg) && (req.Games[1].WinningFlg == req.VictoryFlg)) {
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
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
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
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
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
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
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
					return
				}
			}
		} else {
			// BO1なのに試合数が1つを超えている
			if len(req.Games) > 1 {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}

			if !req.DefaultVictoryFlg && !req.DefaultDefeatFlg {
				// GameのWinningFlgとMatchのVictoryFlgが異なる
				if req.Games[0].WinningFlg != req.VictoryFlg {
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
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
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// RecordIdが空
		if req.RecordId == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultVictoryFlgとDefaultDefeatFlgの両方がtrue
		if req.DefaultVictoryFlg && req.DefaultDefeatFlg {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultVictoryFlgがtrueなのにVictoryFlgがfalse
		if req.DefaultVictoryFlg && !req.VictoryFlg {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultDefeatFlgがtrueなのにVictoryFlgがtrue
		if req.DefaultDefeatFlg && req.VictoryFlg {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		// DefaultVictoryFlg or DefaultDefeatFlgがtrueなのにGamesが存在している
		if req.DefaultVictoryFlg || req.DefaultDefeatFlg {
			if len(req.Games) > 0 {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}
		}

		if req.BO3Flg {
			// BO3なのに試合数が1つ or 3つを超えている
			if len(req.Games) == 1 || len(req.Games) > 3 {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}

			if len(req.Games) == 2 {
				// 試合数が2つの場合はそれぞれのGameのWinningFlgとMatchのVictoryFlgが同じであるべき
				if !((req.Games[0].WinningFlg == req.VictoryFlg) && (req.Games[1].WinningFlg == req.VictoryFlg)) {
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
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
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
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
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
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
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
					return
				}
			}
		} else {
			// BO1なのに試合数が1つを超えている
			if len(req.Games) > 1 {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				ctx.Abort()
				return
			}

			if !req.DefaultVictoryFlg && !req.DefaultDefeatFlg {
				// GameのWinningFlgとMatchのVictoryFlgが異なる
				if req.Games[0].WinningFlg != req.VictoryFlg {
					ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
					ctx.Abort()
					return
				}
			}
		}

		helper.SetMatchUpdateRequest(ctx, req)
	}
}
