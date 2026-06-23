package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type UserStat struct {
	db *gorm.DB
}

func NewUserStat(
	db *gorm.DB,
) repository.UserStatInterface {
	return &UserStat{db}
}

type matchStatsResult struct {
	TotalMatches int
	Wins         int
}

func (i *UserStat) FindUserStat(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (*entity.UserStat, error) {
	var matchResult matchStatsResult

	matchQuery := i.db.Table("matches").
		Select("COUNT(*) AS total_matches, SUM(CASE WHEN victory_flg = true THEN 1 ELSE 0 END) AS wins").
		Where("user_id = ? AND deleted_at IS NULL", userId)

	if !fromDate.IsZero() {
		matchQuery = matchQuery.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		matchQuery = matchQuery.Where("created_at < ?", toDate)
	}

	if tx := matchQuery.Scan(&matchResult); tx.Error != nil {
		return nil, tx.Error
	}

	var recordCount int64
	recordQuery := i.db.Table("records").
		Where("user_id = ? AND deleted_at IS NULL", userId)

	if !fromDate.IsZero() {
		recordQuery = recordQuery.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		recordQuery = recordQuery.Where("created_at < ?", toDate)
	}

	if tx := recordQuery.Count(&recordCount); tx.Error != nil {
		return nil, tx.Error
	}

	var officialEventCount int64
	officialEventQuery := i.db.Table("records").
		Where("user_id = ? AND deleted_at IS NULL AND official_event_id != 0", userId)

	if !fromDate.IsZero() {
		officialEventQuery = officialEventQuery.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		officialEventQuery = officialEventQuery.Where("created_at < ?", toDate)
	}

	if tx := officialEventQuery.Distinct("official_event_id").Count(&officialEventCount); tx.Error != nil {
		return nil, tx.Error
	}

	var tonamelEventCount int64
	tonamelEventQuery := i.db.Table("records").
		Where("user_id = ? AND deleted_at IS NULL AND tonamel_event_id != ''", userId)

	if !fromDate.IsZero() {
		tonamelEventQuery = tonamelEventQuery.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		tonamelEventQuery = tonamelEventQuery.Where("created_at < ?", toDate)
	}

	if tx := tonamelEventQuery.Distinct("tonamel_event_id").Count(&tonamelEventCount); tx.Error != nil {
		return nil, tx.Error
	}

	losses := matchResult.TotalMatches - matchResult.Wins

	var winRate float64
	if matchResult.TotalMatches > 0 {
		winRate = float64(matchResult.Wins) / float64(matchResult.TotalMatches)
	}

	return entity.NewUserStat(userId, int(recordCount), int(officialEventCount), int(tonamelEventCount), matchResult.TotalMatches, matchResult.Wins, losses, winRate), nil
}
