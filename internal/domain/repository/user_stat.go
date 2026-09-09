package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserStatInterface interface {
	FindUserStat(
		ctx context.Context,
		userId string,
		period StatPeriod,
		regulationId uint,
	) (*entity.UserStat, error)
}
