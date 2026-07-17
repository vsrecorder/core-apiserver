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
	//
	// player_id単位の上限は、正当な所有者を巻き込まない値として10回を採用している。
	// 連携フローは verify → (アバター変更) → create と進み1回あたり2回消費するため、
	// 10回はやり直しを含めて5周分にあたる。
	userPlayerAttemptLimiterByUID      = ratelimit.New(10, time.Hour)
	userPlayerAttemptLimiterByPlayerID = ratelimit.New(10, 24*time.Hour)
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

		helper.SetPlayerId(ctx, req.PlayerId)

		if req.ChallengeToken == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !allowUserPlayerAttempt(ctx, req.PlayerId) {
			return
		}

		// player_id の実在確認と所有権(アバター変更チャレンジ)の検証は
		// usecase.Create 内で challenge_token をもとに行うため、ここでは行わない。
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

		helper.SetPlayerId(ctx, req.PlayerId)

		if !allowUserPlayerAttempt(ctx, req.PlayerId) {
			return
		}

		helper.SetUserPlayerVerifyRequest(ctx, req)
	}
}
