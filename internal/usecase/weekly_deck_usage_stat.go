package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type WeeklyDeckUsageStatInterface interface {
	GetWeeklyDeckUsageStat(
		ctx context.Context,
		week string,
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
) (*entity.WeeklyDeckUsageStat, error) {
	// week（週内の任意日 "YYYY-MM-DD"。未指定なら今週）から月曜始まりの週の期間を求める。
	fromDate, toDate, err := weekRange(week, time.Now().Local())
	if err != nil {
		return nil, err
	}

	return u.weeklyDeckUsageStatRepo.FindWeeklyDeckUsageStat(ctx, fromDate, toDate)
}
