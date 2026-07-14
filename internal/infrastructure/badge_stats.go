package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// BadgeStats はバッジ(はじめの一歩・マイルストーン)と週次ストリークの判定に使う集計値を返す。
//
// このリポジトリのクエリは、記録の集計対象外フラグ(ignore_stats_flg)を意図的に見ない。
// 集計対象外は「対戦データの分析(デッキ使用率・対戦相手のデッキ分布・週次メタ・戦績)から
// 除きたい」という意思表示であり、「記録し続けた」という活動量そのものを取り消すものではない。
// バッジ・ストリークは活動量に対する実績のため、集計対象外の記録も一律に数える。
// 分析側(deck_usage_stat/opponent_deck_usage_stat/weekly_deck_usage_stat/user_stat*/
// designation_stats)は従来どおり ignore_stats_flg = false で除外する。
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

func (i *BadgeStats) CountDeckCodesByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("decks").
		Joins("JOIN deck_codes ON deck_codes.deck_id = decks.id AND deck_codes.deleted_at IS NULL AND deck_codes.code IS NOT NULL AND deck_codes.code != ''").
		Where("decks.user_id = ? AND decks.deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("deck_codes.created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("deck_codes.created_at < ?", toDate)
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

func (i *BadgeStats) FindDeckDatesByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) ([]time.Time, error) {
	var dates []time.Time

	query := i.db.Table("decks").Where("user_id = ? AND deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at < ?", toDate)
	}

	if tx := query.Order("created_at ASC").Pluck("created_at", &dates); tx.Error != nil {
		return nil, tx.Error
	}

	return dates, nil
}

func (i *BadgeStats) FindDeckCodeDatesByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) ([]time.Time, error) {
	var dates []time.Time

	query := i.db.Table("decks").
		Joins("JOIN deck_codes ON deck_codes.deck_id = decks.id AND deck_codes.deleted_at IS NULL AND deck_codes.code IS NOT NULL AND deck_codes.code != ''").
		Where("decks.user_id = ? AND decks.deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("deck_codes.created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("deck_codes.created_at < ?", toDate)
	}

	if tx := query.Order("deck_codes.created_at ASC").Pluck("deck_codes.created_at", &dates); tx.Error != nil {
		return nil, tx.Error
	}

	return dates, nil
}

func (i *BadgeStats) FindMatchDatesByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) ([]time.Time, error) {
	var dates []time.Time

	query := i.db.Table("matches").
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL").
		Where("matches.user_id = ? AND matches.deleted_at IS NULL", userId)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	if tx := query.Order("matches.created_at ASC").Pluck("matches.created_at", &dates); tx.Error != nil {
		return nil, tx.Error
	}

	return dates, nil
}
