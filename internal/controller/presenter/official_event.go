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
		date := time.Date(officialEvent.Date.Year(), officialEvent.Date.Month(), officialEvent.Date.Day(), 0, 0, 0, 0, time.Local)

		ret = append(ret, &dto.OfficialEventResponse{
			ID:                      officialEvent.ID,
			Title:                   officialEvent.Title,
			Address:                 officialEvent.Address,
			Venue:                   officialEvent.Venue,
			Date:                    date,
			StartedAt:               officialEvent.StartedAt,
			EndedAt:                 officialEvent.EndedAt,
			TypeName:                officialEvent.TypeName,
			LeagueTitle:             officialEvent.LeagueTitle,
			RegulationTitle:         officialEvent.RegulationTitle,
			CSPFlg:                  officialEvent.CSPFlg,
			Capacity:                officialEvent.Capacity,
			ShopId:                  officialEvent.ShopId,
			ShopName:                officialEvent.ShopName,
			PrefectureId:            officialEvent.PrefectureId,
			PrefectureName:          officialEvent.PrefectureName,
			EnvironmentId:           officialEvent.EnvironmentId,
			EnvironmentTitle:        officialEvent.EnvironmentTitle,
			StandardRegulationId:    officialEvent.StandardRegulationId,
			StandardRegulationMarks: officialEvent.StandardRegulationMarks,
		})
	}

	return &dto.OfficialEventGetResponse{
		TypeId:         typeId,
		LeagueType:     leagueType,
		StartDate:      startDate,
		EndDate:        endDate,
		Count:          count,
		OfficialEvents: ret,
	}
}

func NewOfficialEventGetByIdResponse(
	officialEvent *entity.OfficialEvent,
) *dto.OfficialEventGetByIdResponse {
	date := time.Date(officialEvent.Date.Year(), officialEvent.Date.Month(), officialEvent.Date.Day(), 0, 0, 0, 0, time.Local)

	return &dto.OfficialEventGetByIdResponse{
		OfficialEventResponse: dto.OfficialEventResponse{
			ID:                      officialEvent.ID,
			Title:                   officialEvent.Title,
			Address:                 officialEvent.Address,
			Venue:                   officialEvent.Venue,
			Date:                    date,
			StartedAt:               officialEvent.StartedAt,
			EndedAt:                 officialEvent.EndedAt,
			TypeName:                officialEvent.TypeName,
			LeagueTitle:             officialEvent.LeagueTitle,
			RegulationTitle:         officialEvent.RegulationTitle,
			CSPFlg:                  officialEvent.CSPFlg,
			Capacity:                officialEvent.Capacity,
			ShopId:                  officialEvent.ShopId,
			ShopName:                officialEvent.ShopName,
			PrefectureId:            officialEvent.PrefectureId,
			PrefectureName:          officialEvent.PrefectureName,
			EnvironmentId:           officialEvent.EnvironmentId,
			EnvironmentTitle:        officialEvent.EnvironmentTitle,
			StandardRegulationId:    officialEvent.StandardRegulationId,
			StandardRegulationMarks: officialEvent.StandardRegulationMarks,
		},
	}
}
