package usecase

import (
	"context"
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
		deckId string,
	) ([]*entity.UserStatMonthly, error)
}

type UserStatHistory struct {
	repo                   repository.UserStatHistoryInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
}

func NewUserStatHistory(
	repo repository.UserStatHistoryInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
) UserStatHistoryInterface {
	return &UserStatHistory{repo, championshipSeriesRepo}
}

func (u *UserStatHistory) GetUserStatHistory(
	ctx context.Context,
	userId string,
	period string,
	season string,
	deckId string,
) ([]*entity.UserStatMonthly, error) {
	now := time.Now().Local()
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	nextMonth := thisMonth.AddDate(0, 1, 0)

	var fromDate, toDate time.Time
	var err error

	switch period {
	case "3months":
		fromDate = thisMonth.AddDate(0, -2, 0)
		toDate = nextMonth
	case "6months":
		fromDate = thisMonth.AddDate(0, -5, 0)
		toDate = nextMonth
	default: // "season"
		fromDate, toDate, err = seasonRange(ctx, u.championshipSeriesRepo, season, now)
		if err != nil {
			return nil, err
		}
	}

	return u.repo.FindUserStatHistory(ctx, userId, fromDate, toDate, deckId)
}
