package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserStatHistoryInterface interface {
	FindUserStatHistory(ctx context.Context, userId string, fromDate time.Time, toDate time.Time) ([]*entity.UserStatMonthly, error)
}
