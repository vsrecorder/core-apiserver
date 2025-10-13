package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewOfficialEventGetResponse(
	typeId uint,
	leagueType uint,
	startDate time.Time,
	endDate time.Time,
	count int,
	officialEvents []*entity.OfficialEvent,
) *dto.OfficialEventGetResponse {
	ret := []*dto.OfficialEventResponse{}

	for _, officialEvent := range officialEvents {
		ret = append(ret, &dto.OfficialEventResponse{
			ID:              officialEvent.ID,
			Title:           officialEvent.Title,
			Address:         officialEvent.Address,
			Venue:           officialEvent.Venue,
			Date:            officialEvent.Date.In(time.Local),
			StartedAt:       officialEvent.StartedAt,
			EndedAt:         officialEvent.EndedAt,
			TypeName:        officialEvent.TypeName,
			LeagueTitle:     officialEvent.LeagueTitle,
			RegulationTitle: officialEvent.RegulationTitle,
			CSPFlg:          officialEvent.CSPFlg,
			Capacity:        officialEvent.Capacity,
			ShopId:          officialEvent.ShopId,
			ShopName:        officialEvent.ShopName,
		})
	}

	return &dto.OfficialEventGetResponse{
		TypeId:         typeId,
		LeagueType:     leagueType,
		StartDate:      startDate.In(time.Local),
		EndDate:        endDate.In(time.Local),
		Count:          count,
		OfficialEvents: ret,
	}
}

func NewOfficialEventGetByIdResponse(
	officialEvent *entity.OfficialEvent,
) *dto.OfficialEventGetByIdResponse {
	return &dto.OfficialEventGetByIdResponse{
		OfficialEventResponse: dto.OfficialEventResponse{
			ID:              officialEvent.ID,
			Title:           officialEvent.Title,
			Address:         officialEvent.Address,
			Venue:           officialEvent.Venue,
			Date:            officialEvent.Date.In(time.Local),
			StartedAt:       officialEvent.StartedAt,
			EndedAt:         officialEvent.EndedAt,
			TypeName:        officialEvent.TypeName,
			LeagueTitle:     officialEvent.LeagueTitle,
			RegulationTitle: officialEvent.RegulationTitle,
			CSPFlg:          officialEvent.CSPFlg,
			Capacity:        officialEvent.Capacity,
			ShopId:          officialEvent.ShopId,
			ShopName:        officialEvent.ShopName,
		},
	}
}
