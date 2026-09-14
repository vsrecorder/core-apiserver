package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type WeeklyDeckUsageStatInterface interface {
	GetWeeklyDeckUsageStat(
		ctx context.Context,
		week string,
		grouping entity.DeckUsageGrouping,
	) (*entity.WeeklyDeckUsageStat, error)
}

type WeeklyDeckUsageStat struct {
	weeklyDeckUsageStatRepo repository.WeeklyDeckUsageStatInterface
}

func NewWeeklyDeckUsageStat(
	weeklyDeckUsageStatRepo repository.WeeklyDeckUsageStatInterface,
) WeeklyDeckUsageStatInterface {
	return &WeeklyDeckUsageStat{
		weeklyDeckUsageStatRepo: weeklyDeckUsageStatRepo,
	}
}

func (u *WeeklyDeckUsageStat) GetWeeklyDeckUsageStat(
	ctx context.Context,
	week string,
	grouping entity.DeckUsageGrouping,
) (*entity.WeeklyDeckUsageStat, error) {
	// week（週内の任意日 "YYYY-MM-DD"。未指定なら今週）から月曜始まりの週の期間を求める。
	fromDate, toDate, err := weekRange(week, timeNow().Local())
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// grouping 未指定は既定（スプライトの組み合わせ一致）で集計する。
	if !grouping.IsValid() {
		grouping = entity.DeckUsageGroupingExact
	}

	return u.weeklyDeckUsageStatRepo.FindWeeklyDeckUsageStat(ctx, fromDate, toDate, grouping)
}
