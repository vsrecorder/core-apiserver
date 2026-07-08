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

type Notification struct {
	db *gorm.DB
}

func NewNotification(
	db *gorm.DB,
) repository.NotificationInterface {
	return &Notification{db}
}

func (i *Notification) Save(
	ctx context.Context,
	entity *entity.Notification,
) error {
	m := &model.Notification{
		ID:        entity.ID,
		CreatedAt: entity.CreatedAt,
		UserId:    entity.UserId,
		Category:  entity.Category,
		Title:     entity.Title,
		Body:      entity.Body,
		LinkUrl:   entity.LinkUrl,
		IsRead:    entity.IsRead,
	}
	if !entity.ReadAt.IsZero() {
		m.ReadAt = &entity.ReadAt
	}

	if tx := dbFromContext(ctx, i.db).Create(m); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (i *Notification) UpdateContent(
	ctx context.Context,
	id string,
	createdAt time.Time,
	title string,
	body string,
	isRead bool,
) error {
	tx := dbFromContext(ctx, i.db).Model(&model.Notification{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"created_at": createdAt,
			"title":      title,
			"body":       body,
			"is_read":    isRead,
		})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return apperror.ErrRecordNotFound
	}

	return nil
}

func (i *Notification) FindByUserId(
	ctx context.Context,
	userId string,
	limit int,
) ([]*entity.Notification, error) {
	var models []*model.Notification

	// created_at が同一の通知(1回の評価で複数tier/バッジを同時達成した場合等、
	// 同じachievedAtで複数件作成されうる)でも順序が不定にならないよう、
	// 生成時刻順に単調増加するULIDのidを第2ソートキーとして安定させる。
	// 全体がcreated_at DESC(新しい順)なので、同値時のタイブレークもid DESCにする
	// (例: 称号獲得→直後にランクアップを通知する場合、後発のランク通知の方がidが
	// 大きく、idを新しい順に並べることで称号獲得の通知より上=より新しく表示される)。
	if tx := i.db.
		Where("user_id = ?", userId).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	entities := make([]*entity.Notification, 0, len(models))
	for _, m := range models {
		n := entity.NewNotification(
			m.ID,
			m.CreatedAt,
			m.UserId,
			m.Category,
			m.Title,
			m.Body,
			m.LinkUrl,
		)
		n.IsRead = m.IsRead
		if m.ReadAt != nil {
			n.ReadAt = *m.ReadAt
		}

		entities = append(entities, n)
	}

	return entities, nil
}

func (i *Notification) CountUnreadByUserId(
	ctx context.Context,
	userId string,
) (int, error) {
	var count int64

	if tx := i.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userId, false).
		Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

func (i *Notification) MarkAsRead(
	ctx context.Context,
	id string,
	userId string,
) error {
	now := time.Now().Local()

	tx := i.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userId).
		Updates(map[string]any{
			"is_read": true,
			"read_at": &now,
		})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return apperror.ErrRecordNotFound
	}

	return nil
}

func (i *Notification) MarkAllAsReadByUserId(
	ctx context.Context,
	userId string,
) error {
	now := time.Now().Local()

	tx := i.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userId, false).
		Updates(map[string]any{
			"is_read": true,
			"read_at": &now,
		})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
