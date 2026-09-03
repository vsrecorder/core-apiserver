package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type ChampionsleagueSchedule struct {
	db *gorm.DB
}

func NewChampionsleagueSchedule(
	db *gorm.DB,
) repository.ChampionsleagueScheduleInterface {
	return &ChampionsleagueSchedule{db}
}

func (i *ChampionsleagueSchedule) Find(
	ctx context.Context,
) ([]*entity.ChampionsleagueSchedule, error) {
	var models []*model.ChampionsleagueSchedule

	if tx := i.db.Order("from_date DESC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	entities := []*entity.ChampionsleagueSchedule{}
	for _, model := range models {
		entities = append(entities, entity.NewChampionsleagueSchedule(
			model.ID,
			model.Title,
			model.FromDate,
			model.ToDate,
		))
	}

	return entities, nil
}

func (i *ChampionsleagueSchedule) FindById(
	ctx context.Context,
	id string,
) (*entity.ChampionsleagueSchedule, error) {
	var model model.ChampionsleagueSchedule

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	return entity.NewChampionsleagueSchedule(
		model.ID,
		model.Title,
		model.FromDate,
		model.ToDate,
	), nil
}
