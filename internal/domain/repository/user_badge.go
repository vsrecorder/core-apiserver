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
}
