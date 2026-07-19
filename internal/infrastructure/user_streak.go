package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UserStreak struct {
	db *gorm.DB
}

func NewUserStreak(
	db *gorm.DB,
) repository.UserStreakInterface {
	return &UserStreak{db}
}

func (i *UserStreak) FindByUserId(
	ctx context.Context,
	userId string,
) (*entity.UserStreak, error) {
	var model model.UserStreak

	if tx := i.db.Where("user_id = ?", userId).First(&model); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	return entity.NewUserStreak(
		model.UserId,
		model.CurrentWeeks,
		model.LongestWeeks,
		model.FreezeUsedCount,
		model.FreezeRegenProgress,
		model.LastRecordedWeek,
		model.UpdatedAt,
	), nil
}

func (i *UserStreak) Save(
	ctx context.Context,
	entity *entity.UserStreak,
) error {
	model := &model.UserStreak{
		UserId:              entity.UserId,
		CurrentWeeks:        entity.CurrentWeeks,
		LongestWeeks:        entity.LongestWeeks,
		FreezeUsedCount:     entity.FreezeUsedCount,
		FreezeRegenProgress: entity.FreezeRegenProgress,
		LastRecordedWeek:    entity.LastRecordedWeek,
		UpdatedAt:           entity.UpdatedAt,
	}

	if tx := i.db.Save(model); tx.Error != nil {
		return tx.Error
	}

	return nil
}
