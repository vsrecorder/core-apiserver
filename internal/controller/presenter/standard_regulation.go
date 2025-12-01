package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewStandardRegulationGetResponse(
	standardRegulations []*entity.StandardRegulation,
) []*dto.StandardRegulationResponse {
	ret := []*dto.StandardRegulationResponse{}

	for _, standardRegulation := range standardRegulations {
		fromDate := time.Date(standardRegulation.FromDate.Year(), standardRegulation.FromDate.Month(), standardRegulation.FromDate.Day(), 0, 0, 0, 0, time.Local)
		toDate := time.Date(standardRegulation.ToDate.Year(), standardRegulation.ToDate.Month(), standardRegulation.ToDate.Day(), 0, 0, 0, 0, time.Local)

		ret = append(ret, &dto.StandardRegulationResponse{
			ID:       standardRegulation.ID,
			Marks:    standardRegulation.Marks,
			FromDate: fromDate,
			ToDate:   toDate,
		})
	}

	return ret
}

func NewStandardRegulationGetByIdResponse(
	standardRegulation *entity.StandardRegulation,
) *dto.StandardRegulationResponse {
	fromDate := time.Date(standardRegulation.FromDate.Year(), standardRegulation.FromDate.Month(), standardRegulation.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(standardRegulation.ToDate.Year(), standardRegulation.ToDate.Month(), standardRegulation.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.StandardRegulationResponse{
		ID:       standardRegulation.ID,
		Marks:    standardRegulation.Marks,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}

func NewStandardRegulationGetByMarkResponse(
	standardRegulations []*entity.StandardRegulation,
) []*dto.StandardRegulationResponse {
	ret := []*dto.StandardRegulationResponse{}

	for _, standardRegulation := range standardRegulations {
		fromDate := time.Date(standardRegulation.FromDate.Year(), standardRegulation.FromDate.Month(), standardRegulation.FromDate.Day(), 0, 0, 0, 0, time.Local)
		toDate := time.Date(standardRegulation.ToDate.Year(), standardRegulation.ToDate.Month(), standardRegulation.ToDate.Day(), 0, 0, 0, 0, time.Local)

		ret = append(ret, &dto.StandardRegulationResponse{
			ID:       standardRegulation.ID,
			Marks:    standardRegulation.Marks,
			FromDate: fromDate,
			ToDate:   toDate,
		})
	}

	return ret
}

func NewStandardRegulationGetByDateResponse(
	standardRegulation *entity.StandardRegulation,
) *dto.StandardRegulationResponse {
	fromDate := time.Date(standardRegulation.FromDate.Year(), standardRegulation.FromDate.Month(), standardRegulation.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(standardRegulation.ToDate.Year(), standardRegulation.ToDate.Month(), standardRegulation.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.StandardRegulationResponse{
		ID:       standardRegulation.ID,
		Marks:    standardRegulation.Marks,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
