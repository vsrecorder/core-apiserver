package usecase

import (
	"context"
	"errors"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// pushSubscriptionsPerUserLimit は1ユーザーが同時に持てる生きている購読(端末)の上限。
// 普通の使い方では数台で足りる。endpoint を変えて無制限に登録されると配信バッチが
// 外部へ POST を繰り返す増幅器になるため、ここで打ち切る(既存 endpoint の更新は通す)。
const pushSubscriptionsPerUserLimit = 10

type PushSubscriptionInterface interface {
	// Subscribe は端末の購読を登録する。同じ endpoint なら更新(再購読・持ち主の変更)。
	//
	// 戻り値の wasRevoked は「その endpoint が失効済みとして残っていた」ことを表す。
	// 配信側が 404/410 や連続失敗で失効させた購読は、同じ endpoint で登録し直しても
	// 大抵そのまま死んでいる。端末側はこれを手掛かりに購読を作り直す
	// (端末には購読オブジェクトが残るため、端末だけでは失効に気付けない)。
	Subscribe(
		ctx context.Context,
		userId string,
		endpoint string,
		p256dh string,
		auth string,
		platform string,
	) (wasRevoked bool, err error)

	// Unsubscribe は本人の購読を解除する。既に解除済みでもエラーにしない。
	Unsubscribe(
		ctx context.Context,
		userId string,
		endpoint string,
	) error
}

type PushSubscription struct {
	repository repository.PushSubscriptionInterface
}

func NewPushSubscription(
	repository repository.PushSubscriptionInterface,
) PushSubscriptionInterface {
	return &PushSubscription{repository}
}

func (u *PushSubscription) Subscribe(
	ctx context.Context,
	userId string,
	endpoint string,
	p256dh string,
	auth string,
	platform string,
) (bool, error) {
	live, err := u.repository.FindLiveByUserId(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if len(live) >= pushSubscriptionsPerUserLimit && !hasEndpoint(live, endpoint) {
		return false, apperror.ErrTooManyPushSubscriptions
	}

	// Upsert は revoked_at を消して復活させてしまうため、失効していたかは先に見る。
	// 見つからない(初めての購読)はここでは失敗にしない。
	wasRevoked := false
	if previous, err := u.repository.FindByEndpoint(ctx, endpoint); err == nil {
		wasRevoked = !previous.RevokedAt.IsZero()
	} else if !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return false, err
	}

	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return false, err
	}

	subscription := entity.NewPushSubscription(
		id,
		timeNow(),
		userId,
		endpoint,
		p256dh,
		auth,
		entity.NormalizePushPlatform(platform),
	)

	if err := u.repository.Upsert(ctx, subscription); err != nil {
		logError(ctx, err)
		return false, err
	}

	return wasRevoked, nil
}

func (u *PushSubscription) Unsubscribe(
	ctx context.Context,
	userId string,
	endpoint string,
) error {
	if err := u.repository.RevokeByUserIdAndEndpoint(ctx, userId, endpoint, timeNow()); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func hasEndpoint(subscriptions []*entity.PushSubscription, endpoint string) bool {
	for _, s := range subscriptions {
		if s.Endpoint == endpoint {
			return true
		}
	}

	return false
}
