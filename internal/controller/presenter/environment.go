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
		ret = append(ret, &dto.EnvironmentResponse{
			ID:       environment.ID,
			Title:    environment.Title,
			FromDate: environment.FromDate.In(time.Local),
			ToDate:   environment.ToDate.In(time.Local),
		})
	}

	return ret
}

func NewEnvironmentGetByIdResponse(
	environment *entity.Environment,
) *dto.EnvironmentResponse {
	return &dto.EnvironmentResponse{
		ID:       environment.ID,
		Title:    environment.Title,
		FromDate: environment.FromDate.In(time.Local),
		ToDate:   environment.ToDate.In(time.Local),
	}
}

func NewEnvironmentGetByDateResponse(
	environment *entity.Environment,
) *dto.EnvironmentResponse {
	return &dto.EnvironmentResponse{
		ID:       environment.ID,
		Title:    environment.Title,
		FromDate: environment.FromDate.In(time.Local),
		ToDate:   environment.ToDate.In(time.Local),
	}
}

func NewEnvironmentGetByTermResponse(
	environments []*entity.Environment,
) []*dto.EnvironmentResponse {
	ret := []*dto.EnvironmentResponse{}

	for _, environment := range environments {
		ret = append(ret, &dto.EnvironmentResponse{
			ID:       environment.ID,
			Title:    environment.Title,
			FromDate: environment.FromDate.In(time.Local),
			ToDate:   environment.ToDate.In(time.Local),
		})
	}

	return ret
}
