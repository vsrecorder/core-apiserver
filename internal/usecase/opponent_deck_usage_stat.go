package usecase

import (
	"context"
	"strconv"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type OpponentDeckUsageStatInterface interface {
	GetOpponentDeckUsageStat(
		ctx context.Context,
		userId string,
		yearMonth string,
		environmentId string,
		season string,
		regulationId string,
		deckId string,
	) (*entity.OpponentDeckUsageStat, error)
}

type OpponentDeckUsageStat struct {
	opponentDeckUsageStatRepo repository.OpponentDeckUsageStatInterface
	environmentRepo           repository.EnvironmentInterface
	standardRegulationRepo    repository.StandardRegulationInterface
}

func NewOpponentDeckUsageStat(
	opponentDeckUsageStatRepo repository.OpponentDeckUsageStatInterface,
	environmentRepo repository.EnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
) OpponentDeckUsageStatInterface {
	return &OpponentDeckUsageStat{
		opponentDeckUsageStatRepo: opponentDeckUsageStatRepo,
		environmentRepo:           environmentRepo,
		standardRegulationRepo:    standardRegulationRepo,
	}
}

func (u *OpponentDeckUsageStat) GetOpponentDeckUsageStat(
	ctx context.Context,
	userId string,
	yearMonth string,
	environmentId string,
	season string,
	regulationId string,
	deckId string,
) (*entity.OpponentDeckUsageStat, error) {
	var fromDate, toDate time.Time

	if yearMonth != "" {
		t, err := time.Parse("2006-01", yearMonth)
		if err != nil {
			return nil, err
		}
		fromDate = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
		toDate = fromDate.AddDate(0, 1, 0)
	} else if season != "" {
		// シーズンは9月始まり・翌年8月終わり（例: season=2026 → 2025-09-01 〜 2026-08-31）
		year, err := strconv.Atoi(season)
		if err != nil {
			return nil, err
		}
		fromDate = time.Date(year-1, time.September, 1, 0, 0, 0, 0, time.Local)
		toDate = time.Date(year, time.August, 31, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	}

	if environmentId != "" {
		env, err := u.environmentRepo.FindById(ctx, environmentId)
		if err != nil {
			return nil, err
		}

		envFrom := time.Date(env.FromDate.Year(), env.FromDate.Month(), env.FromDate.Day(), 0, 0, 0, 0, time.Local)
		envTo := time.Date(env.ToDate.Year(), env.ToDate.Month(), env.ToDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

		if fromDate.IsZero() || envFrom.After(fromDate) {
			fromDate = envFrom
		}
		if toDate.IsZero() || envTo.Before(toDate) {
			toDate = envTo
		}
	}

	if regulationId != "" {
		reg, err := u.standardRegulationRepo.FindById(ctx, regulationId)
		if err != nil {
			return nil, err
		}

		regFrom := time.Date(reg.FromDate.Year(), reg.FromDate.Month(), reg.FromDate.Day(), 0, 0, 0, 0, time.Local)
		regTo := time.Date(reg.ToDate.Year(), reg.ToDate.Month(), reg.ToDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

		if fromDate.IsZero() || regFrom.After(fromDate) {
			fromDate = regFrom
		}
		if toDate.IsZero() || regTo.Before(toDate) {
			toDate = regTo
		}
	}

	// いずれも未指定の場合は当月
	if fromDate.IsZero() {
		now := time.Now().Local()
		fromDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		toDate = fromDate.AddDate(0, 1, 0)
	}

	return u.opponentDeckUsageStatRepo.FindOpponentDeckUsageStat(ctx, userId, fromDate, toDate, deckId)
}
