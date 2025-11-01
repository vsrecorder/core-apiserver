package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type CityleagueScheduleInterface interface {
	Find(
		ctx context.Context,
	) ([]*entity.CityleagueSchedule, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.CityleagueSchedule, error)

	FindByDate(
		ctx context.Context,
		date time.Time,
	) (*entity.CityleagueSchedule, error)
}

type CityleagueSchedule struct {
	repository CityleagueScheduleInterface
}

func NewCityleagueSchedule(
	repository CityleagueScheduleInterface,
) *CityleagueSchedule {
	return &CityleagueSchedule{repository}
}

func (u *CityleagueSchedule) Find(
	ctx context.Context,
) ([]*entity.CityleagueSchedule, error) {
	return u.repository.Find(ctx)
}

func (u *CityleagueSchedule) FindById(
	ctx context.Context,
	id string,
) (*entity.CityleagueSchedule, error) {
	return u.repository.FindById(ctx, id)
}

func (u *CityleagueSchedule) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.CityleagueSchedule, error) {
	return u.repository.FindByDate(ctx, date)
}
