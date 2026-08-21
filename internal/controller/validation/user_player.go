package validation

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/ratelimit"
)

// 1人のユーザーが短時間に紐付けを繰り返すのを抑えるためのレート制限。
//
// player_id の実在確認を行わなくなったため、外部サイトへの問い合わせも、他人の
// player_id を総当たりで探索する余地も無い。残しているのは書き込みの連打を防ぐため。
// 通常は1ヶ月ロックがあるため、ここに到達するのは失敗を繰り返した場合に限られる。
var userPlayerAttemptLimiterByUID = ratelimit.New(10, time.Hour)

func UserPlayerCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserPlayerCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if req.PlayerId == "" || len(req.PlayerId) > 16 {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetPlayerId(ctx, req.PlayerId)

		if !userPlayerAttemptLimiterByUID.Allow(helper.GetUID(ctx)) {
			apierror.ErrTooManyRequests.JSON(ctx)
			return
		}

		helper.SetUserPlayerCreateRequest(ctx, req)
	}
}
