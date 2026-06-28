package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type UserStatHistory struct {
	db *gorm.DB
}

func NewUserStatHistory(db *gorm.DB) repository.UserStatHistoryInterface {
	return &UserStatHistory{db}
}

type monthlyMatchResult struct {
	YearMonth    string
	TotalMatches int
	Wins         int
}

func (i *UserStatHistory) FindUserStatHistory(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) ([]*entity.UserStatMonthly, error) {
	var results []monthlyMatchResult

	tx := i.db.Table("matches").
		Select(
			"TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS year_month, "+
				"COUNT(*) AS total_matches, "+
				"SUM(CASE WHEN victory_flg = true THEN 1 ELSE 0 END) AS wins",
		).
		Where("user_id = ? AND deleted_at IS NULL", userId).
		Where("created_at >= ? AND created_at < ?", fromDate, toDate).
		Group("DATE_TRUNC('month', created_at)").
		Order("DATE_TRUNC('month', created_at) ASC").
		Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	history := make([]*entity.UserStatMonthly, 0, len(results))
	for _, r := range results {
		losses := r.TotalMatches - r.Wins
		var winRate float64
		if r.TotalMatches > 0 {
			winRate = float64(r.Wins) / float64(r.TotalMatches)
		}
		history = append(history, entity.NewUserStatMonthly(r.YearMonth, r.TotalMatches, r.Wins, losses, winRate))
	}

	return history, nil
}
