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
		logError(ctx, tx.Error)
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

	if tx := dbFromContext(ctx, i.db).Save(model); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}

// DeleteByUserId は退会時に、そのユーザのストリーク状態を行ごと削除する。
// このテーブルは論理削除を持たないため、残すと参照先の無い行になる。
func (i *UserStreak) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("user_id = ?", uid).Delete(&model.UserStreak{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}
