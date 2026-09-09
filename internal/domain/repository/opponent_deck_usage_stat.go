package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OpponentDeckUsageStatInterface interface {
	FindOpponentDeckUsageStat(
		ctx context.Context,
		userId string,
		period StatPeriod,
		deckId string,
		regulationId uint,
	) (*entity.OpponentDeckUsageStat, error)
}
