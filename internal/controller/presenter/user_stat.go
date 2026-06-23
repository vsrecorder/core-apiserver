package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewUserStatResponse(
	stats *entity.UserStat,
	yearMonth string,
	environmentId string,
	season string,
) *dto.UserStatResponse {
	return &dto.UserStatResponse{
		UserId:             stats.UserId,
		YearMonth:          yearMonth,
		EnvironmentId:      environmentId,
		Season:             season,
		TotalRecords:       stats.TotalRecords,
		OfficialEventCount: stats.OfficialEventCount,
		TonamelEventCount:  stats.TonamelEventCount,
		TotalMatches:       stats.TotalMatches,
		Wins:               stats.Wins,
		Losses:             stats.Losses,
		WinRate:            stats.WinRate,
	}
}
