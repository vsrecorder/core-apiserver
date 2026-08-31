package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// newOfficialEventResponse は公式イベント1件をレスポンス表現へ変換する。
//
// date は DATE カラム由来で時刻を持たないため、ローカル時刻の0時に揃えてから返す
// (DBからは UTC 0時 として読み出されるため、そのままだと日付が1日ずれて見える)。
func newOfficialEventResponse(officialEvent *entity.OfficialEvent) *dto.OfficialEventResponse {
	date := time.Date(officialEvent.Date.Year(), officialEvent.Date.Month(), officialEvent.Date.Day(), 0, 0, 0, 0, time.Local)

	return &dto.OfficialEventResponse{
		ID:                      officialEvent.ID,
		Title:                   officialEvent.Title,
		Address:                 officialEvent.Address,
		Venue:                   officialEvent.Venue,
		Date:                    date,
		StartedAt:               officialEvent.StartedAt,
		EndedAt:                 officialEvent.EndedAt,
		TypeId:                  officialEvent.TypeId,
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
	}
}

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
		ret = append(ret, newOfficialEventResponse(officialEvent))
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
	return &dto.OfficialEventGetByIdResponse{
		OfficialEventResponse: *newOfficialEventResponse(officialEvent),
	}
}
