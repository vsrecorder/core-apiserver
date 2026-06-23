package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserStatInterface interface {
	FindUserStat(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (*entity.UserStat, error)
}
