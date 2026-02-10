package model

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
	TypeId                  uint
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
