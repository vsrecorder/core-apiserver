package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
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

// recordStatsResult は records を1回走査して得る集計値。
// 記録数と、公式/Tonamel/自由形式イベントの種類数を条件付き集計でまとめて数える。
type recordStatsResult struct {
	RecordCount          int
	OfficialEventCount   int
	TonamelEventCount    int
	UnofficialEventCount int
}

func (i *UserStat) FindUserStat(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (*entity.UserStat, error) {
	var matchResult matchStatsResult

	matchQuery := i.db.Table("matches").
		Select("COUNT(*) AS total_matches, SUM(CASE WHEN matches.victory_flg = true THEN 1 ELSE 0 END) AS wins").
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false").
		Where("matches.user_id = ? AND matches.deleted_at IS NULL", userId)

	if !fromDate.IsZero() {
		matchQuery = matchQuery.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		matchQuery = matchQuery.Where("records.event_date < ?", toDate)
	}

	if tx := matchQuery.Scan(&matchResult); tx.Error != nil {
		return nil, tx.Error
	}

	// 記録数と各イベント種類数は、いずれも同じ records の同じ絞り込みに対する集計なので、
	// 記録を4回走査せず1回にまとめる。イベント種類数は「その条件に当てはまる値だけを
	// DISTINCT で数える」ため、CASE で条件外を NULL にして COUNT(DISTINCT ...) に渡す
	// （COUNT は NULL を数えないので、絞り込んでから DISTINCT するのと同じ結果になる）。
	var recordResult recordStatsResult
	recordQuery := i.db.Table("records").
		Select("COUNT(*) AS record_count, "+
			"COUNT(DISTINCT CASE WHEN official_event_id != 0 THEN official_event_id END) AS official_event_count, "+
			"COUNT(DISTINCT CASE WHEN tonamel_event_id != '' THEN tonamel_event_id END) AS tonamel_event_count, "+
			"COUNT(DISTINCT CASE WHEN unofficial_event_id != '' THEN unofficial_event_id END) AS unofficial_event_count").
		Where("user_id = ? AND deleted_at IS NULL AND ignore_stats_flg = false", userId)

	if !fromDate.IsZero() {
		recordQuery = recordQuery.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		recordQuery = recordQuery.Where("event_date < ?", toDate)
	}

	if tx := recordQuery.Scan(&recordResult); tx.Error != nil {
		return nil, tx.Error
	}

	losses := matchResult.TotalMatches - matchResult.Wins

	var winRate float64
	if matchResult.TotalMatches > 0 {
		winRate = float64(matchResult.Wins) / float64(matchResult.TotalMatches)
	}

	return entity.NewUserStat(userId, recordResult.RecordCount, recordResult.OfficialEventCount, recordResult.TonamelEventCount, recordResult.UnofficialEventCount, matchResult.TotalMatches, matchResult.Wins, losses, winRate), nil
}
