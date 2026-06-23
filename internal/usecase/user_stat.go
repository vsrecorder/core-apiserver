package usecase

import (
	"context"
	"strconv"
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
	) (*entity.UserStat, error)
}

type UserStat struct {
	userStatRepo    repository.UserStatInterface
	environmentRepo repository.EnvironmentInterface
}

func NewUserStat(
	userStatRepo repository.UserStatInterface,
	environmentRepo repository.EnvironmentInterface,
) UserStatInterface {
	return &UserStat{
		userStatRepo:    userStatRepo,
		environmentRepo: environmentRepo,
	}
}

func (u *UserStat) GetUserStat(
	ctx context.Context,
	userId string,
	yearMonth string,
	environmentId string,
	season string,
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
		// シーズンは9月始まり・翌年8月終わり（例: season=2026 → 2025-09-01 〜 2026-08-31）
		year, err := strconv.Atoi(season)
		if err != nil {
			return nil, err
		}
		// シーズンは 09-01 始まり・08-31 終わり（exclusive上限は翌シーズン09-01）
		fromDate = time.Date(year-1, time.September, 1, 0, 0, 0, 0, time.Local)
		toDate = time.Date(year, time.August, 31, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
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

	// いずれも未指定の場合は当月
	if fromDate.IsZero() {
		now := time.Now().Local()
		fromDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		toDate = fromDate.AddDate(0, 1, 0)
	}

	return u.userStatRepo.FindUserStat(ctx, userId, fromDate, toDate)
}
