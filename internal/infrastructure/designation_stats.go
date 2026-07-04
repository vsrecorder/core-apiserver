package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// cityLeagueTypeId・trainersLeagueTypeId は official_events.type_id のうち
// シティリーグ・トレーナーズリーグを表す区分(webapp側の officialEventHelpers.ts と同じ判定条件)。
const (
	cityLeagueTypeId     = 2
	trainersLeagueTypeId = 3
)

type DesignationStats struct {
	db *gorm.DB
}

func NewDesignationStats(
	db *gorm.DB,
) repository.DesignationStatsInterface {
	return &DesignationStats{db}
}

func (i *DesignationStats) CountRecordsByUserId(
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

func (i *DesignationStats) CountCityLeagueRecordsByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND official_events.type_id = ?",
			userId, cityLeagueTypeId,
		)
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

func (i *DesignationStats) CountLeagueRecordsByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND official_events.type_id IN (?, ?)",
			userId, cityLeagueTypeId, trainersLeagueTypeId,
		)
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

// userRecordCount は user_id ごとの件数集計クエリの結果を受けるための行構造体。
type userRecordCount struct {
	UserId string
	Count  int
}

func scanUserRecordCounts(query *gorm.DB) (map[string]int, error) {
	var results []userRecordCount

	if tx := query.Scan(&results); tx.Error != nil {
		return nil, tx.Error
	}

	counts := make(map[string]int, len(results))
	for _, r := range results {
		counts[r.UserId] = r.Count
	}

	return counts, nil
}

func (i *DesignationStats) CountRecordsGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("records").
		Select("user_id AS user_id, COUNT(*) AS count").
		Where("deleted_at IS NULL")
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}
	query = query.Group("user_id")

	return scanUserRecordCounts(query)
}

func (i *DesignationStats) CountCityLeagueRecordsGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("records").
		Select("records.user_id AS user_id, COUNT(*) AS count").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.deleted_at IS NULL AND official_events.type_id = ?",
			cityLeagueTypeId,
		)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}
	query = query.Group("records.user_id")

	return scanUserRecordCounts(query)
}

func (i *DesignationStats) CountLeagueRecordsGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("records").
		Select("records.user_id AS user_id, COUNT(*) AS count").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.deleted_at IS NULL AND official_events.type_id IN (?, ?)",
			cityLeagueTypeId, trainersLeagueTypeId,
		)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}
	query = query.Group("records.user_id")

	return scanUserRecordCounts(query)
}
