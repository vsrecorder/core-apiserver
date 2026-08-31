package validation

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// userGymEventsRangeMaxDays はMyジムのイベントを1度に引ける期間の最大日数。
//
// 期間はクライアントが指定するため、上限が無いと1リクエストで登録店舗の全期間を
// 引けてしまう。ホームのパネルは2週間ぶんを出すので、それに対して十分な余裕を取る。
const userGymEventsRangeMaxDays = 62

func UserGymCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.UserGymCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		// shops.id は公式サイト由来の正の整数。0 は「株式会社ポケモン」で店舗ではない。
		if req.ShopId == 0 {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetUserGymCreateRequest(ctx, req)
	}
}

func UserGymDeleteMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		shopId, err := helper.ParseParamShopId(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetShopId(ctx, shopId)
	}
}

func UserGymOfficialEventGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startDate, err := helper.ParseQueryStartDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		endDate, err := helper.ParseQueryEndDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		// startDate > endDate の場合
		if endDate.Before(startDate) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if endDate.Sub(startDate) > userGymEventsRangeMaxDays*24*time.Hour {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetStartDate(ctx, startDate)
		helper.SetEndDate(ctx, endDate)
	}
}
