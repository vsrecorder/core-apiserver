package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func newUserGymResponse(view *entity.UserGymView) *dto.UserGymResponse {
	return &dto.UserGymResponse{
		Shop:      newShopResponse(view.Shop),
		CreatedAt: view.CreatedAt,
	}
}

func newUserGymResponses(views []*entity.UserGymView) []*dto.UserGymResponse {
	ret := []*dto.UserGymResponse{}

	for _, view := range views {
		ret = append(ret, newUserGymResponse(view))
	}

	return ret
}

func NewUserGymGetResponse(
	views []*entity.UserGymView,
) *dto.UserGymGetResponse {
	ret := newUserGymResponses(views)

	return &dto.UserGymGetResponse{
		Limit:    usecase.MaxUserGymsPerUser,
		Count:    len(ret),
		UserGyms: ret,
	}
}

func NewUserGymCreateResponse(
	view *entity.UserGymView,
) *dto.UserGymCreateResponse {
	return &dto.UserGymCreateResponse{
		UserGymResponse: *newUserGymResponse(view),
	}
}

func NewUserGymOfficialEventGetResponse(
	startDate time.Time,
	endDate time.Time,
	views []*entity.UserGymView,
	officialEvents []*entity.OfficialEvent,
) *dto.UserGymOfficialEventGetResponse {
	events := []*dto.OfficialEventResponse{}

	for _, officialEvent := range officialEvents {
		events = append(events, newOfficialEventResponse(officialEvent))
	}

	return &dto.UserGymOfficialEventGetResponse{
		StartDate:      startDate,
		EndDate:        endDate,
		Limit:          usecase.MaxUserGymsPerUser,
		UserGyms:       newUserGymResponses(views),
		Count:          len(events),
		OfficialEvents: events,
	}
}
