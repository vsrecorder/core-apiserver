package model

import "time"

type PlayerRanking struct {
	RankingDate       time.Time `gorm:"primaryKey"`
	LeagueId          int       `gorm:"primaryKey"`
	PlayerId          string    `gorm:"primaryKey"`
	Nickname          string
	CurrentRanking    int
	PrefectureName    string
	ChampionShipPoint int
	PublicFlg         bool
	ChampionFlg       bool
	AvatarImage       string
}

// TableName は実テーブル名 "player_rankings" を明示する。
func (PlayerRanking) TableName() string {
	return "player_rankings"
}
