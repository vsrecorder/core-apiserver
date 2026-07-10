package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type OldestRecord struct {
	db *gorm.DB
}

func NewOldestRecord(
	db *gorm.DB,
) repository.OldestRecordInterface {
	return &OldestRecord{db}
}

type oldestRecordRow struct {
	EventDate *time.Time
}

func (i *OldestRecord) FindOldestRecord(
	ctx context.Context,
	userId string,
	deckId string,
) (*entity.OldestRecord, error) {
	var row oldestRecordRow

	// MIN()は集約関数のため、該当する記録が0件でも常に1行返る
	// （event_dateはSQL NULLになり、*time.Timeとしてnilにスキャンされる）。
	query := i.db.Table("records").
		Select("MIN(event_date) AS event_date").
		Where("user_id = ? AND deleted_at IS NULL AND event_date IS NOT NULL", userId)

	if deckId != "" {
		query = query.Where("deck_id = ?", deckId)
	}

	if tx := query.Scan(&row); tx.Error != nil {
		return nil, tx.Error
	}

	return entity.NewOldestRecord(row.EventDate), nil
}
