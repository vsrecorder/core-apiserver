package entity

import (
	"time"
)

type OfficialEvent struct {
	ID                      uint
	Title                   string
	Address                 string
	Venue                   string
	Date                    time.Time
	StartedAt               time.Time
	EndedAt                 time.Time
	TypeName                string
	LeagueTitle             string
	RegulationTitle         string
	CSPFlg                  bool
	Capacity                uint
	ShopId                  uint
	ShopName                string
	PrefectureId            uint
	PrefectureName          string
	EnvironmentId           string
	EnvironmentTitle        string
	StandardRegulationId    string
	StandardRegulationMarks string
}

func NewOfficialEvent(
	id uint,
	title string,
	address string,
	venue string,
	date time.Time,
	startedAt time.Time,
	endedAt time.Time,
	typeName string,
	leagueTitle string,
	regulationTitle string,
	cspFlg bool,
	capacity uint,
	shopId uint,
	shopName string,
	prefectureId uint,
	prefectureName string,
	environmentId string,
	environmentTitle string,
	standardRegulationId string,
	standardRegulationMarks string,
) *OfficialEvent {
	return &OfficialEvent{
		ID:                      id,
		Title:                   title,
		Address:                 address,
		Venue:                   venue,
		Date:                    date,
		StartedAt:               startedAt,
		EndedAt:                 endedAt,
		TypeName:                typeName,
		LeagueTitle:             leagueTitle,
		RegulationTitle:         regulationTitle,
		CSPFlg:                  cspFlg,
		Capacity:                capacity,
		ShopId:                  shopId,
		ShopName:                shopName,
		PrefectureId:            prefectureId,
		PrefectureName:          prefectureName,
		EnvironmentId:           environmentId,
		EnvironmentTitle:        environmentTitle,
		StandardRegulationId:    standardRegulationId,
		StandardRegulationMarks: standardRegulationMarks,
	}
}
