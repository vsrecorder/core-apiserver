package entity

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

func NewOfficialEvent(
	id uint,
	title string,
	address string,
	venue string,
	date time.Time,
	startedAt time.Time,
	endedAt time.Time,
	deckCount string,
	typeId uint,
	typeName string,
	cspFlg bool,
	leagueId uint,
	leagueTitle string,
	regulationId uint,
	regulationTitle string,
	capacity uint,
	attrId uint,
	shopId uint,
	shopName string,
) *OfficialEvent {
	return &OfficialEvent{
		ID:              id,
		Title:           title,
		Address:         address,
		Venue:           venue,
		Date:            date,
		StartedAt:       startedAt,
		EndedAt:         endedAt,
		DeckCount:       deckCount,
		TypeId:          typeId,
		TypeName:        typeName,
		CSPFlg:          cspFlg,
		LeagueId:        leagueId,
		LeagueTitle:     leagueTitle,
		RegulationId:    regulationId,
		RegulationTitle: regulationTitle,
		Capacity:        capacity,
		AttrId:          attrId,
		ShopId:          shopId,
		ShopName:        shopName,
	}
}
