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
	var result matchStatsResult

	query := i.db.Table("matches").
		Select("COUNT(*) AS total_matches, SUM(CASE WHEN victory_flg = true THEN 1 ELSE 0 END) AS wins").
		Where("user_id = ? AND deleted_at IS NULL", userId)

	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at < ?", toDate)
	}

	if tx := query.Scan(&result); tx.Error != nil {
		return nil, tx.Error
	}

	losses := result.TotalMatches - result.Wins

	var winRate float64
	if result.TotalMatches > 0 {
		winRate = float64(result.Wins) / float64(result.TotalMatches)
	}

	return entity.NewUserStat(userId, result.TotalMatches, result.Wins, losses, winRate), nil
}
