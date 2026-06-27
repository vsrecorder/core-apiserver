package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type DeckUsageStatInterface interface {
	FindDeckUsageStat(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (*entity.DeckUsageStat, error)
}
