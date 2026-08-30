package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type PushSubscriptionInterface interface {
	// Upsert は endpoint をキーに購読を登録・更新する。
	// 同じ端末が再購読すると同じ endpoint が返るため、行を増やさず user_id・鍵・platform を
	// 更新し、revoked_at と failure_count をリセットする(解除後の再購読を生き返らせる)。
	// 既存行の id と created_at は保持される。
	Upsert(
		ctx context.Context,
		entity *entity.PushSubscription,
	) error

	// FindLiveByUserId は解除・失効していない購読を作成順に返す。
	FindLiveByUserId(
		ctx context.Context,
		userId string,
	) ([]*entity.PushSubscription, error)

	// RevokeByUserIdAndEndpoint は本人の購読を解除する。
	// 既に解除済み・存在しない場合もエラーにしない(解除は冪等な操作として扱う)。
	RevokeByUserIdAndEndpoint(
		ctx context.Context,
		userId string,
		endpoint string,
		revokedAt time.Time,
	) error

	// Revoke は配信側の判断(404/410・連続失敗)で購読を失効させる。
	Revoke(
		ctx context.Context,
		id string,
		revokedAt time.Time,
	) error

	// MarkSuccess は配信成功時に failure_count を0へ戻し、last_success_at を更新する。
	MarkSuccess(
		ctx context.Context,
		id string,
		at time.Time,
	) error

	// IncrementFailure は配信失敗時に failure_count を1増やす。
	IncrementFailure(
		ctx context.Context,
		id string,
		at time.Time,
	) error

	// DeleteByUserId は退会時に、そのユーザの購読をまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
