package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type NotificationInterface interface {
	Save(
		ctx context.Context,
		entity *entity.Notification,
	) error

	// UpdateContent は既存通知の内容を上書きする。バックフィルツールが過去に作成した通知を、
	// 再計算結果で上書きするために使う。該当行が無い場合は apperror.ErrRecordNotFound を返す。
	UpdateContent(
		ctx context.Context,
		id string,
		createdAt time.Time,
		title string,
		body string,
		isRead bool,
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

	// FindByUserIdAndCategoryAndBodies は指定ユーザの通知のうち、category が一致し
	// body が bodies のいずれかと完全一致するものを返す。記録の削除・更新で達成条件を
	// 満たさなくなったストリーク継続通知を特定するために使う。通知はバッジ定義への参照を
	// 持たないため、生成時と同じ本文を組み立てて突き合わせる。
	// bodies が空なら空スライスを返す(全件取得にはならない)。
	FindByUserIdAndCategoryAndBodies(
		ctx context.Context,
		userId string,
		category string,
		bodies []string,
	) ([]*entity.Notification, error)

	// DeleteByIds は指定した通知を行ごと削除する。このテーブルは論理削除を持たない。
	// ids が空なら何もしない(全件削除にはならない)。
	DeleteByIds(
		ctx context.Context,
		ids []string,
	) error

	// DeleteByUserId は退会時に、そのユーザの通知をまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
