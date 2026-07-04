package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserStatInterface interface {
	GetUserStat(
		ctx context.Context,
		userId string,
		yearMonth string,
		environmentId string,
		season string,
		regulationId string,
	) (*entity.UserStat, error)
}

type UserStat struct {
	userStatRepo           repository.UserStatInterface
	environmentRepo        repository.EnvironmentInterface
	standardRegulationRepo repository.StandardRegulationInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
}

func NewUserStat(
	userStatRepo repository.UserStatInterface,
	environmentRepo repository.EnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
) UserStatInterface {
	return &UserStat{
		userStatRepo:           userStatRepo,
		environmentRepo:        environmentRepo,
		standardRegulationRepo: standardRegulationRepo,
		championshipSeriesRepo: championshipSeriesRepo,
	}
}

func (u *UserStat) GetUserStat(
	ctx context.Context,
	userId string,
	yearMonth string,
	environmentId string,
	season string,
	regulationId string,
) (*entity.UserStat, error) {
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

		// 環境の期間（to_dateは含む日付なので翌日0時をexclusive上限とする）
		envFrom := time.Date(env.FromDate.Year(), env.FromDate.Month(), env.FromDate.Day(), 0, 0, 0, 0, time.Local)
		envTo := time.Date(env.ToDate.Year(), env.ToDate.Month(), env.ToDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

		// year_month/seasonと環境の両方が指定された場合は期間の交差を取る
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

		// レギュレーションの期間（to_dateは含む日付なので翌日0時をexclusive上限とする）
		regFrom := time.Date(reg.FromDate.Year(), reg.FromDate.Month(), reg.FromDate.Day(), 0, 0, 0, 0, time.Local)
		regTo := time.Date(reg.ToDate.Year(), reg.ToDate.Month(), reg.ToDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

		// 他の条件とレギュレーションの両方が指定された場合は期間の交差を取る
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

	return u.userStatRepo.FindUserStat(ctx, userId, fromDate, toDate)
}
