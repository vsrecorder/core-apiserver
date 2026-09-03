package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func newChampionsleagueScheduleResponse(
	championsleagueSchedule *entity.ChampionsleagueSchedule,
) *dto.ChampionsleagueScheduleResponse {
	fromDate := time.Date(championsleagueSchedule.FromDate.Year(), championsleagueSchedule.FromDate.Month(), championsleagueSchedule.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(championsleagueSchedule.ToDate.Year(), championsleagueSchedule.ToDate.Month(), championsleagueSchedule.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.ChampionsleagueScheduleResponse{
		ID:       championsleagueSchedule.ID,
		Title:    championsleagueSchedule.Title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}

func NewChampionsleagueScheduleGetResponse(
	championsleagueSchedules []*entity.ChampionsleagueSchedule,
) []*dto.ChampionsleagueScheduleResponse {
	ret := []*dto.ChampionsleagueScheduleResponse{}

	for _, championsleagueSchedule := range championsleagueSchedules {
		ret = append(ret, newChampionsleagueScheduleResponse(championsleagueSchedule))
	}

	return ret
}

func NewChampionsleagueScheduleGetByIdResponse(
	championsleagueSchedule *entity.ChampionsleagueSchedule,
) *dto.ChampionsleagueScheduleResponse {
	return newChampionsleagueScheduleResponse(championsleagueSchedule)
}
