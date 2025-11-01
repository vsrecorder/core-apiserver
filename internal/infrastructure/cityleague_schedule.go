package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type CityleagueSchedule struct {
	db *gorm.DB
}

func NewCityleagueSchedule(

	db *gorm.DB,
) repository.CityleagueScheduleInterface {
	return &CityleagueSchedule{db}
}

func (i *CityleagueSchedule) Find(
	ctx context.Context,
) ([]*entity.CityleagueSchedule, error) {
	var models []*model.CityleagueSchedule

	if tx := i.db.Order("from_date DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.CityleagueSchedule
	for _, model := range models {
		entity := entity.NewCityleagueSchedule(
			model.ID,
			model.Title,
			model.FromDate,
			model.ToDate,
		)

		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *CityleagueSchedule) FindById(
	ctx context.Context,
	id string,
) (*entity.CityleagueSchedule, error) {
	var model model.CityleagueSchedule

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewCityleagueSchedule(
		model.ID,
		model.Title,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}

func (i *CityleagueSchedule) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.CityleagueSchedule, error) {
	var model model.CityleagueSchedule

	if tx := i.db.Where("from_date <= ? AND to_date >= ?", date, date).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewCityleagueSchedule(
		model.ID,
		model.Title,
		model.FromDate,
		model.ToDate,
	)

	return entity, nil
}
