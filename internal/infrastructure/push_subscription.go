package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type PushSubscription struct {
	db *gorm.DB
}

func NewPushSubscription(
	db *gorm.DB,
) repository.PushSubscriptionInterface {
	return &PushSubscription{db}
}

func newPushSubscriptionEntity(m *model.PushSubscription) *entity.PushSubscription {
	e := entity.NewPushSubscription(
		m.ID,
		m.CreatedAt,
		m.UserId,
		m.Endpoint,
		m.P256dh,
		m.Auth,
		m.Platform,
	)
	e.UpdatedAt = m.UpdatedAt
	e.FailureCount = m.FailureCount
	if m.RevokedAt != nil {
		e.RevokedAt = *m.RevokedAt
	}
	if m.LastSuccessAt != nil {
		e.LastSuccessAt = *m.LastSuccessAt
	}

	return e
}

// Upsert は endpoint をキーに購読を登録・更新する。
// 同じ端末の再購読では行を増やさず、鍵と持ち主を更新して revoked_at / failure_count を
// リセットする(既存行の id と created_at はそのまま)。
func (i *PushSubscription) Upsert(
	ctx context.Context,
	entity *entity.PushSubscription,
) error {
	m := &model.PushSubscription{
		ID:        entity.ID,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
		UserId:    entity.UserId,
		Endpoint:  entity.Endpoint,
		P256dh:    entity.P256dh,
		Auth:      entity.Auth,
		Platform:  entity.Platform,
	}

	tx := dbFromContext(ctx, i.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"updated_at":    entity.UpdatedAt,
			"revoked_at":    nil,
			"user_id":       entity.UserId,
			"p256dh":        entity.P256dh,
			"auth":          entity.Auth,
			"platform":      entity.Platform,
			"failure_count": 0,
		}),
	}).Create(m)
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	return nil
}

func (i *PushSubscription) FindLiveByUserId(
	ctx context.Context,
	userId string,
) ([]*entity.PushSubscription, error) {
	var models []*model.PushSubscription

	tx := dbFromContext(ctx, i.db).
		Where("user_id = ? AND revoked_at IS NULL", userId).
		Order("created_at ASC").
		Find(&models)
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	entities := make([]*entity.PushSubscription, 0, len(models))
	for _, m := range models {
		entities = append(entities, newPushSubscriptionEntity(m))
	}

	return entities, nil
}

func (i *PushSubscription) RevokeByUserIdAndEndpoint(
	ctx context.Context,
	userId string,
	endpoint string,
	revokedAt time.Time,
) error {
	tx := dbFromContext(ctx, i.db).Model(&model.PushSubscription{}).
		Where("user_id = ? AND endpoint = ? AND revoked_at IS NULL", userId, endpoint).
		Updates(map[string]any{
			"revoked_at": revokedAt,
			"updated_at": revokedAt,
		})
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	// 解除は冪等な操作として扱い、該当行が無くてもエラーにしない
	return nil
}

func (i *PushSubscription) Revoke(
	ctx context.Context,
	id string,
	revokedAt time.Time,
) error {
	tx := dbFromContext(ctx, i.db).Model(&model.PushSubscription{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{
			"revoked_at": revokedAt,
			"updated_at": revokedAt,
		})
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	return nil
}

func (i *PushSubscription) MarkSuccess(
	ctx context.Context,
	id string,
	at time.Time,
) error {
	tx := dbFromContext(ctx, i.db).Model(&model.PushSubscription{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"failure_count":   0,
			"last_success_at": at,
			"updated_at":      at,
		})
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	return nil
}

func (i *PushSubscription) IncrementFailure(
	ctx context.Context,
	id string,
	at time.Time,
) error {
	tx := dbFromContext(ctx, i.db).Model(&model.PushSubscription{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"failure_count": gorm.Expr("failure_count + 1"),
			"updated_at":    at,
		})
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	return nil
}

// DeleteByUserId は退会時に、そのユーザの購読を行ごと削除する。
// このテーブルは論理削除を持たないため、残すと参照先の無い行になる。
func (i *PushSubscription) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("user_id = ?", uid).Delete(&model.PushSubscription{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}
