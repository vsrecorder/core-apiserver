package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type WeeklyDeckUsageStatInterface interface {
	FindWeeklyDeckUsageStat(
		ctx context.Context,
		fromDate time.Time,
		toDate time.Time,
	) (*entity.WeeklyDeckUsageStat, error)
}
