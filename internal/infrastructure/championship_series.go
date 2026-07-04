package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type ChampionshipSeries struct {
	db *gorm.DB
}

func NewChampionshipSeries(
	db *gorm.DB,
) repository.ChampionshipSeriesInterface {
	return &ChampionshipSeries{db}
}

func (i *ChampionshipSeries) Find(
	ctx context.Context,
) ([]*entity.ChampionshipSeries, error) {
	var models []*model.ChampionshipSeries

	if tx := i.db.Order("from_date DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.ChampionshipSeries
	for _, model := range models {
		entity := entity.NewChampionshipSeries(
			model.ID,
			model.Title,
			model.FromDate,
			model.ToDate,
		)

		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *ChampionshipSeries) FindById(
	ctx context.Context,
	id string,
) (*entity.ChampionshipSeries, error) {
	var model model.ChampionshipSeries

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	entity := entity.NewChampionshipSeries(
		model.ID,
		model.Title,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}

func (i *ChampionshipSeries) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.ChampionshipSeries, error) {
	var model model.ChampionshipSeries

	if tx := i.db.Where("from_date <= ? AND to_date >= ?", date, date).First(&model); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	entity := entity.NewChampionshipSeries(
		model.ID,
		model.Title,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}
