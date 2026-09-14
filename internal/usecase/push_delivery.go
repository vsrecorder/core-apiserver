package usecase

import (
	"context"
	"errors"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
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
	// あわせて、その push のもとになったアプリ内通知を既読にする。
	MarkClicked(
		ctx context.Context,
		userId string,
		id string,
	) error
}

type PushDelivery struct {
	repository             repository.PushDeliveryInterface
	notificationRepository repository.NotificationInterface
}

func NewPushDelivery(
	repository repository.PushDeliveryInterface,
	notificationRepository repository.NotificationInterface,
) PushDeliveryInterface {
	return &PushDelivery{repository, notificationRepository}
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

	u.markNotificationAsRead(ctx, userId, id)

	return nil
}

/*
 * タップして開いた push のもとになったアプリ内通知を既読にする。
 *
 * push を開いた時点で本人はその知らせを見ているので、ベルに未読が残っていると
 * 「読んだのに消えない」ことになる。配達ログは notification_id を持っているので、
 * タップの記録と同じ経路でたどれる。
 *
 * 失敗しても MarkClicked は成功として返す。ここでの目的はタップの記録であり、
 * 既読化はその副次的な後始末だから。未読が残るだけで、ベルから開けば既読にできる。
 * 通知が見つからない場合(古い配達ログ・通知を伴わない push・既に削除済み)も同じ扱いにする。
 */
func (u *PushDelivery) markNotificationAsRead(
	ctx context.Context,
	userId string,
	deliveryId string,
) {
	delivery, err := u.repository.FindById(ctx, deliveryId, userId)
	if err != nil {
		if !errors.Is(err, apperror.ErrRecordNotFound) {
			logError(ctx, err)
		}
		return
	}

	if delivery.NotificationId == "" {
		return
	}

	if err := u.notificationRepository.MarkAsRead(ctx, delivery.NotificationId, userId); err != nil {
		if !errors.Is(err, apperror.ErrRecordNotFound) {
			logError(ctx, err)
		}
	}
}
