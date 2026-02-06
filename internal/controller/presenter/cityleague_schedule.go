package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewCityleagueScheduleGetResponse(
	cityleagueSchedules []*entity.CityleagueSchedule,
) []*dto.CityleagueScheduleResponse {
	ret := []*dto.CityleagueScheduleResponse{}

	for _, cityleagueSchedule := range cityleagueSchedules {
		fromDate := time.Date(cityleagueSchedule.FromDate.Year(), cityleagueSchedule.FromDate.Month(), cityleagueSchedule.FromDate.Day(), 0, 0, 0, 0, time.Local)
		toDate := time.Date(cityleagueSchedule.ToDate.Year(), cityleagueSchedule.ToDate.Month(), cityleagueSchedule.ToDate.Day(), 0, 0, 0, 0, time.Local)

		ret = append(ret, &dto.CityleagueScheduleResponse{
			ID:       cityleagueSchedule.ID,
			Title:    cityleagueSchedule.Title,
			FromDate: fromDate,
			ToDate:   toDate,
		})
	}

	return ret
}

func NewCityleagueScheduleGetByIdResponse(
	cityleagueSchedule *entity.CityleagueSchedule,
) *dto.CityleagueScheduleResponse {
	fromDate := time.Date(cityleagueSchedule.FromDate.Year(), cityleagueSchedule.FromDate.Month(), cityleagueSchedule.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(cityleagueSchedule.ToDate.Year(), cityleagueSchedule.ToDate.Month(), cityleagueSchedule.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.CityleagueScheduleResponse{
		ID:       cityleagueSchedule.ID,
		Title:    cityleagueSchedule.Title,
		FromDate: fromDate,
		ToDate:   toDate,
	}

}

func NewCityleagueScheduleGetByDateResponse(
	cityleagueSchedule *entity.CityleagueSchedule,
) *dto.CityleagueScheduleResponse {
	fromDate := time.Date(cityleagueSchedule.FromDate.Year(), cityleagueSchedule.FromDate.Month(), cityleagueSchedule.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(cityleagueSchedule.ToDate.Year(), cityleagueSchedule.ToDate.Month(), cityleagueSchedule.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.CityleagueScheduleResponse{
		ID:       cityleagueSchedule.ID,
		Title:    cityleagueSchedule.Title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
