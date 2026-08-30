package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type PushDelivery struct {
	db *gorm.DB
}

func NewPushDelivery(
	db *gorm.DB,
) repository.PushDeliveryInterface {
	return &PushDelivery{db}
}

func newPushDeliveryEntity(m *model.PushDelivery) *entity.PushDelivery {
	e := entity.NewPushDelivery(
		m.ID,
		m.CreatedAt,
		m.UserId,
		m.SubscriptionId,
		m.NotificationId,
		m.Campaign,
		m.Status,
		m.StatusCode,
	)
	if m.DeliveredAt != nil {
		e.DeliveredAt = *m.DeliveredAt
	}
	if m.ClickedAt != nil {
		e.ClickedAt = *m.ClickedAt
	}

	return e
}

func (i *PushDelivery) Save(
	ctx context.Context,
	entity *entity.PushDelivery,
) error {
	m := &model.PushDelivery{
		ID:             entity.ID,
		CreatedAt:      entity.CreatedAt,
		UserId:         entity.UserId,
		SubscriptionId: entity.SubscriptionId,
		NotificationId: entity.NotificationId,
		Campaign:       entity.Campaign,
		Status:         entity.Status,
		StatusCode:     entity.StatusCode,
	}
	if !entity.DeliveredAt.IsZero() {
		m.DeliveredAt = &entity.DeliveredAt
	}
	if !entity.ClickedAt.IsZero() {
		m.ClickedAt = &entity.ClickedAt
	}

	if tx := dbFromContext(ctx, i.db).Create(m); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}

func (i *PushDelivery) UpdateResult(
	ctx context.Context,
	id string,
	status string,
	statusCode int,
) error {
	tx := dbFromContext(ctx, i.db).Model(&model.PushDelivery{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      status,
			"status_code": statusCode,
		})
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return apperror.ErrRecordNotFound
	}

	return nil
}

// markAt は列を「未設定なら at、設定済みならそのまま」に更新する。
// user_id を条件に含めることで、本人以外の配達ログは更新されず(RowsAffected=0)、
// 存在しない場合と同じく apperror.ErrRecordNotFound になる。
// COALESCE で最初の時刻を保つため、同じ端末が2回送っても行は更新される(冪等)。
func (i *PushDelivery) markAt(
	ctx context.Context,
	column string,
	id string,
	userId string,
	at time.Time,
) error {
	tx := dbFromContext(ctx, i.db).Model(&model.PushDelivery{}).
		Where("id = ? AND user_id = ?", id, userId).
		Update(column, gorm.Expr("COALESCE("+column+", ?)", at))
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return apperror.ErrRecordNotFound
	}

	return nil
}

func (i *PushDelivery) MarkDelivered(
	ctx context.Context,
	id string,
	userId string,
	at time.Time,
) error {
	return i.markAt(ctx, "delivered_at", id, userId, at)
}

func (i *PushDelivery) MarkClicked(
	ctx context.Context,
	id string,
	userId string,
	at time.Time,
) error {
	return i.markAt(ctx, "clicked_at", id, userId, at)
}

func (i *PushDelivery) CountNotificationsByUserIdAndCampaignsSince(
	ctx context.Context,
	userId string,
	campaigns []string,
	since time.Time,
) (int, error) {
	if len(campaigns) == 0 {
		return 0, nil
	}

	// 失敗・失効した配達は数えない。金曜の送出が全端末で失敗した週に日曜の nudge まで
	// 止めてしまうと、何も届いていないのに上限だけが効く
	var count int64
	tx := dbFromContext(ctx, i.db).Model(&model.PushDelivery{}).
		Where("user_id = ? AND campaign IN ? AND status = ? AND created_at >= ?", userId, campaigns, entity.PushDeliveryStatusSent, since).
		Distinct("notification_id").
		Count(&count)
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return 0, wrapError(tx.Error)
	}

	return int(count), nil
}

func (i *PushDelivery) FindRecentByUserIdAndCampaign(
	ctx context.Context,
	userId string,
	campaign string,
	limit int,
) ([]*entity.PushDelivery, error) {
	var models []*model.PushDelivery

	tx := dbFromContext(ctx, i.db).
		Where("user_id = ? AND campaign = ?", userId, campaign).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&models)
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	entities := make([]*entity.PushDelivery, 0, len(models))
	for _, m := range models {
		entities = append(entities, newPushDeliveryEntity(m))
	}

	return entities, nil
}

// DeleteByUserId は退会時に、そのユーザの配達ログを行ごと削除する。
// このテーブルは論理削除を持たないため、残すと参照先の無い行になる。
func (i *PushDelivery) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("user_id = ?", uid).Delete(&model.PushDelivery{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}
