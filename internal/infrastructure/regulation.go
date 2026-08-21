package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type Regulation struct {
	db *gorm.DB
}

func NewRegulation(
	db *gorm.DB,
) repository.RegulationInterface {
	return &Regulation{db}
}

func (i *Regulation) Find(
	ctx context.Context,
) ([]*entity.Regulation, error) {
	var models []*model.Regulation

	// 選択UIの並び順をDB側で固定する(スタンダード→エクストラ→殿堂)。
	if tx := i.db.Order("id ASC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	var entities []*entity.Regulation
	for _, model := range models {
		entity := entity.NewRegulation(
			model.ID,
			model.Name,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}
