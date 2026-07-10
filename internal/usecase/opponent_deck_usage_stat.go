package usecase

import (
	"context"
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
	championshipSeriesRepo    repository.ChampionshipSeriesInterface
}

func NewOpponentDeckUsageStat(
	opponentDeckUsageStatRepo repository.OpponentDeckUsageStatInterface,
	environmentRepo repository.EnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
) OpponentDeckUsageStatInterface {
	return &OpponentDeckUsageStat{
		opponentDeckUsageStatRepo: opponentDeckUsageStatRepo,
		environmentRepo:           environmentRepo,
		standardRegulationRepo:    standardRegulationRepo,
		championshipSeriesRepo:    championshipSeriesRepo,
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
		var err error
		fromDate, toDate, err = seasonRange(ctx, u.championshipSeriesRepo, season, time.Now().Local())
		if err != nil {
			return nil, err
		}
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

	// yearMonth/season/environmentId/regulationIdのいずれも未指定の場合は、
	// fromDate/toDateをゼロ値のまま渡し「全期間」として扱う
	// （repository側はゼロ値の場合event_dateによる絞り込みを行わない）
	return u.opponentDeckUsageStatRepo.FindOpponentDeckUsageStat(ctx, userId, fromDate, toDate, deckId)
}
