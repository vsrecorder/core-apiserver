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
		ret = append(ret, &dto.CityleagueScheduleResponse{
			ID:       cityleagueSchedule.ID,
			Title:    cityleagueSchedule.Title,
			FromDate: cityleagueSchedule.FromDate.In(time.Local),
			ToDate:   cityleagueSchedule.ToDate.In(time.Local),
		})
	}

	return ret
}

func NewCityleagueScheduleGetByIdResponse(
	cityleagueSchedule *entity.CityleagueSchedule,
) *dto.CityleagueScheduleResponse {
	return &dto.CityleagueScheduleResponse{
		ID:       cityleagueSchedule.ID,
		Title:    cityleagueSchedule.Title,
		FromDate: cityleagueSchedule.FromDate.In(time.Local),
		ToDate:   cityleagueSchedule.ToDate.In(time.Local),
	}

}

func NewCityleagueScheduleGetByDateResponse(
	cityleagueSchedule *entity.CityleagueSchedule,
) *dto.CityleagueScheduleResponse {
	return &dto.CityleagueScheduleResponse{
		ID:       cityleagueSchedule.ID,
		Title:    cityleagueSchedule.Title,
		FromDate: cityleagueSchedule.FromDate.In(time.Local),
		ToDate:   cityleagueSchedule.ToDate.In(time.Local),
	}
}
