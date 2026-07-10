package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type OldestRecordInterface interface {
	GetOldestRecord(
		ctx context.Context,
		userId string,
		deckId string,
	) (*entity.OldestRecord, error)
}

type OldestRecord struct {
	oldestRecordRepo repository.OldestRecordInterface
}

func NewOldestRecord(
	oldestRecordRepo repository.OldestRecordInterface,
) OldestRecordInterface {
	return &OldestRecord{
		oldestRecordRepo: oldestRecordRepo,
	}
}

func (u *OldestRecord) GetOldestRecord(
	ctx context.Context,
	userId string,
	deckId string,
) (*entity.OldestRecord, error) {
	return u.oldestRecordRepo.FindOldestRecord(ctx, userId, deckId)
}
