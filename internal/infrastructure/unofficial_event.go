package infrastructure

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type UnofficialEvent struct {
	db *gorm.DB
}

func NewUnofficialEvent(
	db *gorm.DB,
) repository.UnofficialEventInterface {
	return &UnofficialEvent{db}
}

func (i *UnofficialEvent) FindById(
	ctx context.Context,
	id string,
) (*entity.UnofficialEvent, error) {
	var model model.UnofficialEvent

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewUnofficialEvent(
		model.ID,
		model.UserId,
		model.Title,
		model.Date,
	)

	return entity, nil
}

func (i *UnofficialEvent) Save(
	ctx context.Context,
	entity *entity.UnofficialEvent,
) error {
	model := model.NewUnofficialEvent(
		entity.ID,
		entity.UserId,
		entity.Title,
		entity.Date,
	)

	if tx := i.db.Save(model); tx.Error != nil {
		return tx.Error
	}

	return nil
}
