package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type Environment struct {
	db *gorm.DB
}

func NewEnvironment(
	db *gorm.DB,
) repository.EnvironmentInterface {
	return &Environment{db}
}

func (i *Environment) Find(
	ctx context.Context,
) ([]*entity.Environment, error) {
	var models []*model.Environment

	if tx := i.db.Order("from_date DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Environment
	for _, model := range models {
		entity := entity.NewEnvironment(
			model.ID,
			model.Title,
			model.FromDate,
			model.ToDate,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Environment) FindById(
	ctx context.Context,
	id string,
) (*entity.Environment, error) {
	var model *model.Environment

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewEnvironment(
		model.ID,
		model.Title,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}

func (i *Environment) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.Environment, error) {
	var model *model.Environment

	if tx := i.db.Where("from_date <= ?", date).Order("from_date DESC").First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewEnvironment(
		model.ID,
		model.Title,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}

func (i *Environment) FindByTerm(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) ([]*entity.Environment, error) {
	var models []*model.Environment

	if tx := i.db.Where("to_date >= ? AND from_date <= ?", fromDate, toDate).Order("from_date DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Environment
	for _, model := range models {
		entity := entity.NewEnvironment(
			model.ID,
			model.Title,
			model.FromDate,
			model.ToDate,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}
