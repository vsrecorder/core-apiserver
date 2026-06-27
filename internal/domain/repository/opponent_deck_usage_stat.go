package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OpponentDeckUsageStatInterface interface {
	FindOpponentDeckUsageStat(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (*entity.OpponentDeckUsageStat, error)
}
