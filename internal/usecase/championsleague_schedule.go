package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type ChampionsleagueScheduleInterface interface {
	Find(
		ctx context.Context,
	) ([]*entity.ChampionsleagueSchedule, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.ChampionsleagueSchedule, error)
}

type ChampionsleagueSchedule struct {
	repository ChampionsleagueScheduleInterface
}

func NewChampionsleagueSchedule(
	repository ChampionsleagueScheduleInterface,
) *ChampionsleagueSchedule {
	return &ChampionsleagueSchedule{repository}
}

func (u *ChampionsleagueSchedule) Find(
	ctx context.Context,
) ([]*entity.ChampionsleagueSchedule, error) {
	return u.repository.Find(ctx)
}

func (u *ChampionsleagueSchedule) FindById(
	ctx context.Context,
	id string,
) (*entity.ChampionsleagueSchedule, error) {
	return u.repository.FindById(ctx, id)
}
