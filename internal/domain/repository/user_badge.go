package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserBadgeInterface interface {
	FindByUserId(
		ctx context.Context,
		userId string,
	) ([]*entity.UserBadge, error)

	Save(
		ctx context.Context,
		entity *entity.UserBadge,
	) error

	// DeleteByUserId は退会時に、そのユーザの獲得バッジをまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
