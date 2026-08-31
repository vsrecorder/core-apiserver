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

	// ExistsByUserIdAndCategoryAndLinkUrl は指定ユーザに category と link_url が一致する通知が
	// 1件でもあるかを返す。週次レポートのように「同じリンク先の通知は1回だけ」という
	// 冪等性を、直近N件に頼らず全期間で判定するために使う。
	ExistsByUserIdAndCategoryAndLinkUrl(
		ctx context.Context,
		userId string,
		category string,
		linkUrl string,
	) (bool, error)

	// DeleteByUserId は退会時に、そのユーザの通知をまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
