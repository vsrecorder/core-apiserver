package usecase

import (
	"context"
	"strconv"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserStatHistoryInterface interface {
	GetUserStatHistory(
		ctx context.Context,
		userId string,
		period string,
		season string,
	) ([]*entity.UserStatMonthly, error)
}

type UserStatHistory struct {
	repo repository.UserStatHistoryInterface
}

func NewUserStatHistory(repo repository.UserStatHistoryInterface) UserStatHistoryInterface {
	return &UserStatHistory{repo}
}

func (u *UserStatHistory) GetUserStatHistory(
	ctx context.Context,
	userId string,
	period string,
	season string,
) ([]*entity.UserStatMonthly, error) {
	now := time.Now().Local()
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	nextMonth := thisMonth.AddDate(0, 1, 0)

	var fromDate, toDate time.Time

	switch period {
	case "3months":
		fromDate = thisMonth.AddDate(0, -2, 0)
		toDate = nextMonth
	case "6months":
		fromDate = thisMonth.AddDate(0, -5, 0)
		toDate = nextMonth
	default: // "season"
		var seasonYear int
		if season != "" {
			var err error
			seasonYear, err = strconv.Atoi(season)
			if err != nil {
				return nil, err
			}
		} else {
			if now.Month() >= time.September {
				seasonYear = now.Year() + 1
			} else {
				seasonYear = now.Year()
			}
		}
		fromDate = time.Date(seasonYear-1, time.September, 1, 0, 0, 0, 0, time.Local)
		toDate = time.Date(seasonYear, time.August, 31, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	}

	return u.repo.FindUserStatHistory(ctx, userId, fromDate, toDate)
}
