package presenter

import (
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
			FromDate: environment.FromDate,
			ToDate:   environment.ToDate,
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
		FromDate: environment.FromDate,
		ToDate:   environment.ToDate,
	}
}

func NewEnvironmentGetByDateResponse(
	environment *entity.Environment,
) *dto.EnvironmentResponse {
	return &dto.EnvironmentResponse{
		ID:       environment.ID,
		Title:    environment.Title,
		FromDate: environment.FromDate,
		ToDate:   environment.ToDate,
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
			FromDate: environment.FromDate,
			ToDate:   environment.ToDate,
		})
	}

	return ret
}
