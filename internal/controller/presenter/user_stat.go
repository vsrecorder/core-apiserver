package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewUserStatResponse(
	stats *entity.UserStat,
	week string,
	yearMonth string,
	environmentId string,
	season string,
	standardRegulationId string,
	regulationId uint,
) *dto.UserStatResponse {
	return &dto.UserStatResponse{
		UserId:               stats.UserId,
		Week:                 week,
		YearMonth:            yearMonth,
		EnvironmentId:        environmentId,
		Season:               season,
		StandardRegulationId: standardRegulationId,
		RegulationId:         regulationId,
		TotalRecords:         stats.TotalRecords,
		OfficialEventCount:   stats.OfficialEventCount,
		TonamelEventCount:    stats.TonamelEventCount,
		UnofficialEventCount: stats.UnofficialEventCount,
		TotalMatches:         stats.TotalMatches,
		Wins:                 stats.Wins,
		Losses:               stats.Losses,
		WinRate:              stats.WinRate,
	}
}
