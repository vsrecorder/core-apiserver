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
	Draws        int
}

func (i *UserStatHistory) FindUserStatHistory(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
	deckId string,
	regulationId uint,
) ([]*entity.UserStatMonthly, error) {
	var results []monthlyMatchResult

	query := i.db.Table("matches").
		Select(
			"TO_CHAR(DATE_TRUNC('month', records.event_date), 'YYYY-MM') AS year_month, "+
				"COUNT(*) AS total_matches, "+
				"SUM(CASE WHEN matches.victory_flg = true THEN 1 ELSE 0 END) AS wins, "+
				"SUM(CASE WHEN matches.draw_flg = true THEN 1 ELSE 0 END) AS draws",
		).
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false").
		Where("matches.user_id = ? AND matches.deleted_at IS NULL", userId).
		Where("records.event_date >= ? AND records.event_date < ?", fromDate, toDate)

	if deckId != "" {
		// デッキセレクタは records.deck_id を基準に選択肢を作っている（deck_usage_stat.go参照）。
		// matches.deck_id はマッチ作成時点の値がコピーされたまま更新されないため、
		// 記録後にデッキを変更した対戦がズレて、選択肢に出ているデッキの推移が
		// 実際より少なく（別のデッキでは多く）出てしまう。
		// そのため records.deck_id で絞り込み、選択肢と実データの基準を一致させる。
		query = query.Where("records.deck_id = ?", deckId)
	}

	// レギュレーション(スタンダード/エクストラ/殿堂)での絞り込み。0 は絞り込みなし。
	if regulationId != 0 {
		query = query.Where("records.regulation_id = ?", regulationId)
	}

	tx := query.
		Group("DATE_TRUNC('month', records.event_date)").
		Order("DATE_TRUNC('month', records.event_date) ASC").
		Scan(&results)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	history := make([]*entity.UserStatMonthly, 0, len(results))
	for _, r := range results {
		// 引き分けは負けに数えない。勝率も分母から除外する(勝ち/(勝ち+負け))。
		losses := r.TotalMatches - r.Wins - r.Draws
		var winRate float64
		if decided := r.Wins + losses; decided > 0 {
			winRate = float64(r.Wins) / float64(decided)
		}
		history = append(history, entity.NewUserStatMonthly(r.YearMonth, r.TotalMatches, r.Wins, losses, winRate))
	}

	return history, nil
}
