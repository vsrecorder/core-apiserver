package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type NotificationInterface interface {
	ListByUserId(
		ctx context.Context,
		userId string,
		limit int,
	) ([]*entity.Notification, error)

	CountUnreadByUserId(
		ctx context.Context,
		userId string,
	) (int, error)

	MarkAsRead(
		ctx context.Context,
		userId string,
		id string,
	) error

	MarkAllAsRead(
		ctx context.Context,
		userId string,
	) error
}

type Notification struct {
	repository repository.NotificationInterface
}

func NewNotification(
	repository repository.NotificationInterface,
) NotificationInterface {
	return &Notification{repository}
}

func (u *Notification) ListByUserId(
	ctx context.Context,
	userId string,
	limit int,
) ([]*entity.Notification, error) {
	return u.repository.FindByUserId(ctx, userId, limit)
}

func (u *Notification) CountUnreadByUserId(
	ctx context.Context,
	userId string,
) (int, error) {
	return u.repository.CountUnreadByUserId(ctx, userId)
}

func (u *Notification) MarkAsRead(
	ctx context.Context,
	userId string,
	id string,
) error {
	return u.repository.MarkAsRead(ctx, id, userId)
}

func (u *Notification) MarkAllAsRead(
	ctx context.Context,
	userId string,
) error {
	return u.repository.MarkAllAsReadByUserId(ctx, userId)
}
