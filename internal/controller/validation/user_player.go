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
	// 実在確認そのものは webapp(BFF)が行い、外部サイトへ問い合わせる前に webapp 側でも
	// 同様の制限をかけている。ここでの制限は、webapp を経由しない直接のリクエストに
	// 対しても紐付けの試行回数を抑えるための多層防御。
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

		if req.VerificationToken == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !allowUserPlayerAttempt(ctx, req.PlayerId) {
			return
		}

		// 所有権の確認は webapp が済ませており、その結果である検証済みトークンの
		// 署名検証は usecase.Create 内で行うため、ここでは行わない。
		helper.SetUserPlayerCreateRequest(ctx, req)
	}
}

func UserPlayerChallengeMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserPlayerChallengeRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// current_avatar_image は除外条件にすぎず、空でも「どのアバターでもよい」
		// という意味になるため必須にはしない。

		helper.SetUserPlayerChallengeRequest(ctx, req)
	}
}
