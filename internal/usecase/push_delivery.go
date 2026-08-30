package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type PushDeliveryInterface interface {
	// MarkDelivered は端末の Service Worker が push を受け取ったことを記録する。
	// 本人の配達ログ以外は apperror.ErrRecordNotFound。
	MarkDelivered(
		ctx context.Context,
		userId string,
		id string,
	) error

	// MarkClicked は通知がタップされ、リンク先が開かれたことを記録する。
	MarkClicked(
		ctx context.Context,
		userId string,
		id string,
	) error
}

type PushDelivery struct {
	repository repository.PushDeliveryInterface
}

func NewPushDelivery(
	repository repository.PushDeliveryInterface,
) PushDeliveryInterface {
	return &PushDelivery{repository}
}

func (u *PushDelivery) MarkDelivered(
	ctx context.Context,
	userId string,
	id string,
) error {
	if err := u.repository.MarkDelivered(ctx, id, userId, timeNow()); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func (u *PushDelivery) MarkClicked(
	ctx context.Context,
	userId string,
	id string,
) error {
	if err := u.repository.MarkClicked(ctx, id, userId, timeNow()); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}
