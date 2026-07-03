package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UserBadge struct {
	db *gorm.DB
}

func NewUserBadge(
	db *gorm.DB,
) repository.UserBadgeInterface {
	return &UserBadge{db}
}

func (i *UserBadge) FindByUserId(
	ctx context.Context,
	userId string,
) ([]*entity.UserBadge, error) {
	var models []*model.UserBadge

	if tx := i.db.Where("user_id = ?", userId).Order("achieved_at ASC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.UserBadge
	for _, model := range models {
		entities = append(entities, entity.NewUserBadge(
			model.ID,
			model.CreatedAt,
			model.UserId,
			model.BadgeDefinitionId,
			model.RecordId,
			model.AchievedAt,
		))
	}

	return entities, nil
}

func (i *UserBadge) Save(
	ctx context.Context,
	entity *entity.UserBadge,
) error {
	model := &model.UserBadge{
		ID:                entity.ID,
		CreatedAt:         entity.CreatedAt,
		UserId:            entity.UserId,
		BadgeDefinitionId: entity.BadgeDefinitionId,
		RecordId:          entity.RecordId,
		AchievedAt:        entity.AchievedAt,
	}

	if tx := i.db.Create(model); tx.Error != nil {
		return tx.Error
	}

	return nil
}
