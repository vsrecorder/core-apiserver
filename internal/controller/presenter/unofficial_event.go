package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewUnofficialEventGetByIdResponse(
	unofficialEvent *entity.UnofficialEvent,
) *dto.UnofficialEventGetByIdResponse {
	date := time.Date(unofficialEvent.Date.Year(), unofficialEvent.Date.Month(), unofficialEvent.Date.Day(), 0, 0, 0, 0, time.Local)

	return &dto.UnofficialEventGetByIdResponse{
		UnofficialEventResponse: dto.UnofficialEventResponse{
			ID:     unofficialEvent.ID,
			UserId: unofficialEvent.UserId,
			Title:  unofficialEvent.Title,
			Date:   date,
		},
	}
}

func NewUnofficialEventCreateResponse(
	unofficialEvent *entity.UnofficialEvent,
) *dto.UnofficialEventCreateResponse {
	date := time.Date(unofficialEvent.Date.Year(), unofficialEvent.Date.Month(), unofficialEvent.Date.Day(), 0, 0, 0, 0, time.Local)

	return &dto.UnofficialEventCreateResponse{
		UnofficialEventResponse: dto.UnofficialEventResponse{
			ID:     unofficialEvent.ID,
			UserId: unofficialEvent.UserId,
			Title:  unofficialEvent.Title,
			Date:   date,
		},
	}
}
