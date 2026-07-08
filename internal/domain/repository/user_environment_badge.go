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
}
