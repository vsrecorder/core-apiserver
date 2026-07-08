package infrastructure

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// Save は user_id, badge_definition_id の組み合わせで既に行が存在する場合は
// 何もしない(DoNothing)。呼び出し元(award()やbackfillツール)が未達成のバッジのみを
// 渡す設計だが、リトライや再実行で同じ組み合わせが二重に渡されても一意制約違反で
// 失敗しないようにするため、Save自体を冪等にする。
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

	tx := i.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "badge_definition_id"}},
		DoNothing: true,
	}).Create(model)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
