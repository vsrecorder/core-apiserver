package entity

import (
	"time"
)

// LeagueId は公式サイトのリーグID。リーグのシリーズごとに振られ、league_title が同じでも
// シリーズが違えば別の値になる(例: トレーナーズリーグ2026 はオープン=1410・ジュニア=1413・
// シニア=1414・マスター=1417)。DB では NULL があり得て、その場合は 0 になる。
type OfficialEvent struct {
	ID                      uint
	Title                   string
	Address                 string
	Venue                   string
	Date                    time.Time
	StartedAt               time.Time
	EndedAt                 time.Time
	TypeId                  uint
	TypeName                string
	LeagueId                uint
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
	typeId uint,
	typeName string,
	leagueId uint,
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
		TypeId:                  typeId,
		TypeName:                typeName,
		LeagueId:                leagueId,
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
