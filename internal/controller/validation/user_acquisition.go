package validation

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/ratelimit"
)

// 流入元は1ユーザーにつき登録直後の1回しか送られない(2回目以降は保存側が無視する)。
// リトライぶんの余裕だけ見て、それ以上の連打は書き込みの試行ごと止める。
var userAcquisitionLimiterByUID = ratelimit.New(10, time.Hour)

func UserAcquisitionCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserAcquisitionCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if !userAcquisitionLimiterByUID.Allow(helper.GetUID(ctx)) {
			apierror.ErrTooManyRequests.JSON(ctx)
			return
		}

		helper.SetUserAcquisitionCreateRequest(ctx, req)
	}
}

// アンケートも登録直後の1回しか送られない(2回目以降は保存側が初回を優先する)。
// リトライぶんの余裕だけ見て、それ以上の連打は止める。
var userAcquisitionSurveyLimiterByUID = ratelimit.New(10, time.Hour)

func UserAcquisitionSurveyMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserAcquisitionSurveyRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		// 回答はUIの4択からしか来ない。allowlist 外を黙って捨てるとUI側の実装ミスに
		// 気づけないため、流入元(値の中身は下流で丸める)と違ってここで 400 にする。
		if entity.NormalizeAcquisitionSurveyAnswer(req.Answer) == "" {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if !userAcquisitionSurveyLimiterByUID.Allow(helper.GetUID(ctx)) {
			apierror.ErrTooManyRequests.JSON(ctx)
			return
		}

		helper.SetUserAcquisitionSurveyRequest(ctx, req)
	}
}
