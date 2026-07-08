package infrastructure

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UserEnvironmentBadge struct {
	db *gorm.DB
}

func NewUserEnvironmentBadge(
	db *gorm.DB,
) repository.UserEnvironmentBadgeInterface {
	return &UserEnvironmentBadge{db}
}

func (i *UserEnvironmentBadge) FindByUserId(
	ctx context.Context,
	userId string,
) ([]*entity.UserEnvironmentBadge, error) {
	var models []*model.UserEnvironmentBadge

	if tx := i.db.Where("user_id = ?", userId).Order("achieved_at ASC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.UserEnvironmentBadge
	for _, model := range models {
		entities = append(entities, entity.NewUserEnvironmentBadge(
			model.UserId,
			model.EnvironmentId,
			model.RecordId,
			model.NotificationId,
			model.AchievedAt,
			model.CreatedAt,
		))
	}

	return entities, nil
}

// Save は user_id, environment_id の組み合わせで upsert する。既に行がある場合は
// achieved_at, created_at のみを新しい値で上書きし、record_id/notification_id は
// 最初に作成された時点の値のまま変更しない(バックフィルツールの再実行で、判定基準の
// 変更に応じて達成日時だけを再計算・上書きできるようにするため)。
func (i *UserEnvironmentBadge) Save(
	ctx context.Context,
	entity *entity.UserEnvironmentBadge,
) error {
	model := &model.UserEnvironmentBadge{
		UserId:         entity.UserId,
		EnvironmentId:  entity.EnvironmentId,
		RecordId:       entity.RecordId,
		NotificationId: entity.NotificationId,
		AchievedAt:     entity.AchievedAt,
		CreatedAt:      entity.CreatedAt,
	}

	tx := dbFromContext(ctx, i.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "environment_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"achieved_at", "created_at"}),
	}).Create(model)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
