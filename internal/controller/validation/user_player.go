package validation

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/ratelimit"
)

var (
	// 他人の player_id を大量に試行する、いわゆる総当たりを抑止するためのレート制限。
	// uid単位: 1人のユーザーが短時間に多数の player_id を試すのを防ぐ。
	// player_id単位: 複数アカウントを使って特定の player_id を繰り返し狙うのを防ぐ。
	userPlayerAttemptLimiterByUID      = ratelimit.New(10, time.Hour)
	userPlayerAttemptLimiterByPlayerID = ratelimit.New(5, 24*time.Hour)
)

func allowUserPlayerAttempt(ctx *gin.Context, playerId string) bool {
	uid := helper.GetUID(ctx)

	if !userPlayerAttemptLimiterByUID.Allow(uid) {
		apierror.ErrTooManyRequests.JSON(ctx)
		return false
	}

	if !userPlayerAttemptLimiterByPlayerID.Allow(playerId) {
		apierror.ErrTooManyRequests.JSON(ctx)
		return false
	}

	return true
}

func UserPlayerCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserPlayerCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.PlayerId == "" || len(req.PlayerId) > 16 {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !allowUserPlayerAttempt(ctx, req.PlayerId) {
			return
		}

		checkPlayerId(ctx, req.PlayerId)
		if ctx.IsAborted() {
			return
		}

		helper.SetUserPlayerCreateRequest(ctx, req)
	}
}

func UserPlayerVerifyMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserPlayerVerifyRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if req.PlayerId == "" || len(req.PlayerId) > 16 {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !allowUserPlayerAttempt(ctx, req.PlayerId) {
			return
		}

		helper.SetUserPlayerVerifyRequest(ctx, req)
	}
}
