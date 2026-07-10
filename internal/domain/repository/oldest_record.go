package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OldestRecordInterface interface {
	FindOldestRecord(
		ctx context.Context,
		userId string,
		deckId string,
	) (*entity.OldestRecord, error)
}
