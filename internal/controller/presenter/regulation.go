package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewRegulationGetResponse(
	regulations []*entity.Regulation,
) []*dto.RegulationResponse {
	ret := []*dto.RegulationResponse{}

	for _, regulation := range regulations {
		ret = append(ret, &dto.RegulationResponse{
			ID:   regulation.ID,
			Name: regulation.Name,
		})
	}

	return ret
}
