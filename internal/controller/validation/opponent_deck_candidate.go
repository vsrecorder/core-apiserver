package validation

import (
	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// OpponentDeckCandidateGetMiddleware は GET /matches/opponent_deck_candidates の limit を検証する。
// 上限は helper.MaxLimit(100)で、usecase が保持する候補数と同じ。
func OpponentDeckCandidateGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := helper.ParseQueryLimit(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetLimit(ctx, limit)
	}
}
