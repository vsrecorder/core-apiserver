package validation

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

const (
	// 「直近N戦」で受け付ける件数。画面のセレクタと同じく20〜100戦の10戦刻み。
	// 未指定時は最小の20戦(画面の初期表示と揃える)。
	recentMatchCountMin  = 20
	recentMatchCountMax  = 100
	recentMatchCountStep = 10
)

func UserStatRecentGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		countStr := helper.GetQueryCount(ctx)
		count := recentMatchCountMin
		if countStr != "" {
			n, err := strconv.Atoi(countStr)
			if err != nil ||
				n < recentMatchCountMin ||
				n > recentMatchCountMax ||
				n%recentMatchCountStep != 0 {
				apierror.ErrBadRequest.JSON(ctx)
				return
			}
			count = n
		}
		helper.SetLimit(ctx, count)

		helper.SetDeckId(ctx, helper.GetQueryDeckId(ctx))

		// レギュレーション区分(スタンダード/エクストラ/殿堂)での絞り込み
		helper.SetRegulationId(ctx, helper.ParseQueryRegulationId(ctx))
	}
}
