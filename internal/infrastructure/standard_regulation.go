package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type StandardRegulation struct {
	db *gorm.DB
}

func NewStandardRegulation(
	db *gorm.DB,
) repository.StandardRegulationInterface {
	return &StandardRegulation{db}
}

func (i *StandardRegulation) Find(
	ctx context.Context,
) ([]*entity.StandardRegulation, error) {
	var models []*model.StandardRegulation

	if tx := i.db.Order("from_date DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.StandardRegulation
	for _, model := range models {
		entity := entity.NewStandardRegulation(
			model.ID,
			model.Marks,
			model.FromDate,
			model.ToDate,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *StandardRegulation) FindById(
	ctx context.Context,
	id string,
) (*entity.StandardRegulation, error) {
	var model model.StandardRegulation

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewStandardRegulation(
		model.ID,
		model.Marks,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}

func (i *StandardRegulation) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.StandardRegulation, error) {
	var model model.StandardRegulation

	if tx := i.db.Where("from_date <= ? AND to_date >= ?", date, date).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewStandardRegulation(
		model.ID,
		model.Marks,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}
