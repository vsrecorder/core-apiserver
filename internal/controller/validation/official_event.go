package validation

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// officialEventsRangeMaxDays は公式イベントを1度に引ける期間の最大日数。
//
// 期間は呼び出し側が指定し、ログインなしで呼べる。official_events は100万行を超えており、
// 上限が無いと全期間を1回指定されるだけで、その全行に店舗・環境などを結合して並べ替える
// ことになる。件数の上限(infrastructure の officialEventsFindLimit)は返す量を抑えるが、
// DB が読む量は抑えないので、期間の側でも絞る。
//
// 正規の利用で最も長いのは、webapp のシティリーグ結果(直近90日ぶん)と環境ごとの一覧
// (環境の長さ。最長は83日)。環境がこれより長くなっても収まるよう、約2倍の余裕を取る。
const officialEventsRangeMaxDays = 186

func OfficialEventGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		typeId, err := helper.ParseQueryTypeId(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		date, err := helper.ParseQueryDate(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

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
		if !startDate.Before(endDate) && !startDate.Equal(endDate) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		if endDate.Sub(startDate) > officialEventsRangeMaxDays*24*time.Hour {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetTypeId(ctx, typeId)
		helper.SetLeagueType(ctx, leagueType)
		helper.SetDate(ctx, date)
		helper.SetStartDate(ctx, startDate)
		helper.SetEndDate(ctx, endDate)
	}
}

func OfficialEventGetByIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := helper.GetId(ctx)

		officialEventId, err := strconv.Atoi(id)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		} else if officialEventId <= 0 {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetOfficialEventId(ctx, uint(officialEventId))
	}
}
