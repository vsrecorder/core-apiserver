package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewEnvironmentGetResponse(
	environments []*entity.Environment,
) []*dto.EnvironmentResponse {
	ret := []*dto.EnvironmentResponse{}

	for _, environment := range environments {
		fromDate := time.Date(environment.FromDate.Year(), environment.FromDate.Month(), environment.FromDate.Day(), 0, 0, 0, 0, time.Local)
		toDate := time.Date(environment.ToDate.Year(), environment.ToDate.Month(), environment.ToDate.Day(), 0, 0, 0, 0, time.Local)

		ret = append(ret, &dto.EnvironmentResponse{
			ID:       environment.ID,
			Title:    environment.Title,
			FromDate: fromDate,
			ToDate:   toDate,
		})
	}

	return ret
}

func NewEnvironmentGetByIdResponse(
	environment *entity.Environment,
) *dto.EnvironmentResponse {
	fromDate := time.Date(environment.FromDate.Year(), environment.FromDate.Month(), environment.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(environment.ToDate.Year(), environment.ToDate.Month(), environment.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.EnvironmentResponse{
		ID:       environment.ID,
		Title:    environment.Title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}

func NewEnvironmentGetByDateResponse(
	environment *entity.Environment,
) *dto.EnvironmentResponse {
	fromDate := time.Date(environment.FromDate.Year(), environment.FromDate.Month(), environment.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(environment.ToDate.Year(), environment.ToDate.Month(), environment.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.EnvironmentResponse{
		ID:       environment.ID,
		Title:    environment.Title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}

func NewEnvironmentGetByTermResponse(
	environments []*entity.Environment,
) []*dto.EnvironmentResponse {
	ret := []*dto.EnvironmentResponse{}

	for _, environment := range environments {
		fromDate := time.Date(environment.FromDate.Year(), environment.FromDate.Month(), environment.FromDate.Day(), 0, 0, 0, 0, time.Local)
		toDate := time.Date(environment.ToDate.Year(), environment.ToDate.Month(), environment.ToDate.Day(), 0, 0, 0, 0, time.Local)

		ret = append(ret, &dto.EnvironmentResponse{
			ID:       environment.ID,
			Title:    environment.Title,
			FromDate: fromDate,
			ToDate:   toDate,
		})
	}

	return ret
}
