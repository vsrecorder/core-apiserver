package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type PlayerRanking struct {
	db *gorm.DB
}

func NewPlayerRanking(
	db *gorm.DB,
) repository.PlayerRankingInterface {
	return &PlayerRanking{db}
}

func (i *PlayerRanking) FindLatestByPlayerId(
	ctx context.Context,
	playerId string,
) (*entity.PlayerRanking, error) {
	var m model.PlayerRanking

	if tx := i.db.Where("player_id = ?", playerId).Order("ranking_date DESC").First(&m); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	return entity.NewPlayerRanking(
		m.PlayerId,
		m.RankingDate,
		m.ChampionShipPoint,
	), nil
}
