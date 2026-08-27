package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserEnvironmentBadgeInterface interface {
	FindByUserId(
		ctx context.Context,
		userId string,
	) ([]*entity.UserEnvironmentBadge, error)

	Save(
		ctx context.Context,
		entity *entity.UserEnvironmentBadge,
	) error

	// DeleteByUserId は退会時に、そのユーザの獲得環境バッジをまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
