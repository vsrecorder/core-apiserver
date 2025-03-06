package model

import (
	"time"
)

type OfficialEvent struct {
	ID              uint
	Title           string
	Address         string
	Venue           string
	Date            time.Time
	StartedAt       time.Time
	EndedAt         time.Time
	DeckCount       string
	TypeId          uint
	TypeName        string
	CSPFlg          bool
	LeagueId        uint
	LeagueTitle     string
	RegulationId    uint
	RegulationTitle string
	Capacity        uint
	AttrId          uint
	ShopId          uint
	ShopName        string
}
