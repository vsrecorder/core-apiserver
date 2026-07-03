package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type BadgeDefinition struct {
	db *gorm.DB
}

func NewBadgeDefinition(
	db *gorm.DB,
) repository.BadgeDefinitionInterface {
	return &BadgeDefinition{db}
}

func (i *BadgeDefinition) FindAll(
	ctx context.Context,
) ([]*entity.BadgeDefinition, error) {
	var models []*model.BadgeDefinition

	if tx := i.db.Order("created_at ASC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.BadgeDefinition
	for _, model := range models {
		entities = append(entities, entity.NewBadgeDefinition(
			model.ID,
			model.Code,
			model.Category,
			model.Name,
			model.Description,
			model.IconKey,
			model.CriteriaType,
			model.CriteriaValue,
			model.AvailableFrom,
			model.AvailableTo,
			model.CreatedAt,
			model.UpdatedAt,
		))
	}

	return entities, nil
}
