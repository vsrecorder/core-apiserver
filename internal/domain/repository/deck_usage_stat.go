package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type DeckUsageStatInterface interface {
	FindDeckUsageStat(
		ctx context.Context,
		userId string,
		period StatPeriod,
		regulationId uint,
	) (*entity.DeckUsageStat, error)
}
