package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type Designation struct {
	db *gorm.DB
}

func NewDesignation(
	db *gorm.DB,
) repository.DesignationInterface {
	return &Designation{db}
}

func (i *Designation) FindAll(
	ctx context.Context,
) ([]*entity.Designation, error) {
	var models []*model.Designation

	if tx := i.db.Order("tier ASC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Designation
	for _, model := range models {
		entities = append(entities, entity.NewDesignation(
			model.ID,
			model.Tier,
			model.Code,
			model.Emoji,
			model.Name,
			model.Description,
			model.CriteriaType,
			model.CriteriaValue,
			model.CreatedAt,
			model.UpdatedAt,
		))
	}

	return entities, nil
}
