package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
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
	deckId string,
) ([]*entity.UserStatMonthly, error) {
	var results []monthlyMatchResult

	query := i.db.Table("matches").
		Select(
			"TO_CHAR(DATE_TRUNC('month', records.event_date), 'YYYY-MM') AS year_month, "+
				"COUNT(*) AS total_matches, "+
				"SUM(CASE WHEN matches.victory_flg = true THEN 1 ELSE 0 END) AS wins",
		).
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false").
		Where("matches.user_id = ? AND matches.deleted_at IS NULL", userId).
		Where("records.event_date >= ? AND records.event_date < ?", fromDate, toDate)

	if deckId != "" {
		query = query.Where("matches.deck_id = ?", deckId)
	}

	tx := query.
		Group("DATE_TRUNC('month', records.event_date)").
		Order("DATE_TRUNC('month', records.event_date) ASC").
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
