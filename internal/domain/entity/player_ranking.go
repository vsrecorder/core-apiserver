package entity

import "time"

type PlayerRanking struct {
	PlayerId          string
	RankingDate       time.Time
	ChampionShipPoint int
}

func NewPlayerRanking(
	playerId string,
	rankingDate time.Time,
	championShipPoint int,
) *PlayerRanking {
	return &PlayerRanking{
		PlayerId:          playerId,
		RankingDate:       rankingDate,
		ChampionShipPoint: championShipPoint,
	}
}
