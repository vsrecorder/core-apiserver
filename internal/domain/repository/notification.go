package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type NotificationInterface interface {
	Save(
		ctx context.Context,
		entity *entity.Notification,
	) error

	// FindByUserId は指定ユーザーの通知を created_at 降順で最大 limit 件返す。
	FindByUserId(
		ctx context.Context,
		userId string,
		limit int,
	) ([]*entity.Notification, error)

	CountUnreadByUserId(
		ctx context.Context,
		userId string,
	) (int, error)

	// MarkAsRead は userId 本人の通知のみを既読にする。該当行が無い場合は
	// apperror.ErrRecordNotFound を返す。
	MarkAsRead(
		ctx context.Context,
		id string,
		userId string,
	) error

	MarkAllAsReadByUserId(
		ctx context.Context,
		userId string,
	) error
}
