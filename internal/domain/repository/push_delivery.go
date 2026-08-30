package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type PushDeliveryInterface interface {
	Save(
		ctx context.Context,
		entity *entity.PushDelivery,
	) error

	// UpdateResult は送出結果(status / status_code)を書き込む。
	// 配達ログは送出前に pending で作り、プッシュサービスの応答でこれを呼ぶ。
	UpdateResult(
		ctx context.Context,
		id string,
		status string,
		statusCode int,
	) error

	// MarkDelivered は端末が push を受け取ったことを記録する。
	// 本人の配達ログだけを更新する(user_id を条件に含める)ため、他人の id を指定しても
	// 更新されず apperror.ErrRecordNotFound を返す。2回目以降は最初の時刻を保つ。
	MarkDelivered(
		ctx context.Context,
		id string,
		userId string,
		at time.Time,
	) error

	// MarkClicked は通知がタップされたことを記録する。所有者の扱いは MarkDelivered と同じ。
	MarkClicked(
		ctx context.Context,
		id string,
		userId string,
		at time.Time,
	) error

	// CountNotificationsByUserIdAndCampaignsSince は since 以降に campaigns で push が受理された
	// (status = sent)通知の数(notification_id の distinct。端末数ではなく通知数)を返す。
	// 「1ユーザー週2通まで」の判定に使う。失敗・失効した配達は届いていないので数えない。
	CountNotificationsByUserIdAndCampaignsSince(
		ctx context.Context,
		userId string,
		campaigns []string,
		since time.Time,
	) (int, error)

	// FindRecentByUserIdAndCampaign は campaign の配達ログを新しい順に最大 limit 件返す。
	// 反応の無いユーザーへの配信を間引く判定に使う。
	FindRecentByUserIdAndCampaign(
		ctx context.Context,
		userId string,
		campaign string,
		limit int,
	) ([]*entity.PushDelivery, error)

	// DeleteByUserId は退会時に、そのユーザの配達ログをまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
