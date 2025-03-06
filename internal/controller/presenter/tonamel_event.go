package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewTonamelEventGetByIdResponse(
	tonamelEvent *entity.TonamelEvent,
) *dto.TonamelEventGetByIdResponse {
	return &dto.TonamelEventGetByIdResponse{
		TonamelEventResponse: dto.TonamelEventResponse{
			ID:          tonamelEvent.ID,
			Title:       tonamelEvent.Title,
			Description: tonamelEvent.Description,
			Image:       tonamelEvent.Image,
		},
	}
}
