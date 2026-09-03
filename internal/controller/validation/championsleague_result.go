package validation

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// championsleagueScheduleIdMaxLength は championsleague_schedules.id の桁数(VARCHAR(63))。
// これを超える値は該当し得ないため、DBを引く前に弾く。
const championsleagueScheduleIdMaxLength = 63

// ChampionsleagueResultGetByChampionsleagueScheduleIdMiddleware は大会IDとリーグ区分の検証。
//
// シティリーグと違い期間や日付では絞り込ませない。大型大会は年に数回しか開催されず、
// 「直近7日」のような既定の期間では空振りするうえ、閲覧の単位が大会(スケジュール)だから。
// そのため大会IDは必須にしている。
// league_type は任意で、未指定(0)なら全リーグ区分が対象になる。
func ChampionsleagueResultGetByChampionsleagueScheduleIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		championsleagueScheduleId := helper.GetQueryChampionsleagueScheduleId(ctx)

		if championsleagueScheduleId == "" {
			apierror.ErrBadRequest.JSON(ctx, errors.New("championsleague_schedule_id is required"))
			return
		}

		if len(championsleagueScheduleId) > championsleagueScheduleIdMaxLength {
			apierror.ErrBadRequest.JSON(ctx, errors.New("championsleague_schedule_id is too long"))
			return
		}

		leagueType, err := helper.ParseQueryLeagueType(ctx)
		if err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		helper.SetChampionsleagueScheduleId(ctx, championsleagueScheduleId)
		helper.SetLeagueType(ctx, leagueType)
	}
}
