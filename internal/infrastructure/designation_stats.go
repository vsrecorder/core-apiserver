package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// gymBattleTypeId は official_events.type_id のうち「ジムバトル」を含む区分。
// type_id=4 は他の区分(MEGAウインターリーグ等)も混在するため、title に「ジムバトル」を
// 含むかどうかも合わせて判定する。webapp側の officialEventHelpers.ts と同じ判定条件。
const gymBattleTypeId = 4

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

func (i *DesignationStats) CountGymBattleRecordsByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND official_events.type_id = ? AND official_events.title LIKE ?",
			userId, gymBattleTypeId, "%ジムバトル%",
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
