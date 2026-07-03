package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type BadgeStats struct {
	db *gorm.DB
}

func NewBadgeStats(
	db *gorm.DB,
) repository.BadgeStatsInterface {
	return &BadgeStats{db}
}

func (i *BadgeStats) CountRecordsByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").Where("user_id = ? AND deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}

	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

func (i *BadgeStats) CountMatchesByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("matches").
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL").
		Where("matches.user_id = ? AND matches.deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

func (i *BadgeStats) CountDecksByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("decks").Where("user_id = ? AND deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at < ?", toDate)
	}

	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

func (i *BadgeStats) FindRecordDatesByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) ([]time.Time, error) {
	type recordDate struct {
		EventDate time.Time
		CreatedAt time.Time
	}
	var rows []recordDate

	query := i.db.Table("records").
		Select("event_date, created_at").
		Where("user_id = ? AND deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("COALESCE(event_date, created_at) >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("COALESCE(event_date, created_at) < ?", toDate)
	}

	if tx := query.Scan(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	dates := make([]time.Time, 0, len(rows))
	for _, r := range rows {
		basis := r.EventDate
		if basis.IsZero() {
			basis = r.CreatedAt
		}
		dates = append(dates, basis)
	}

	return dates, nil
}
