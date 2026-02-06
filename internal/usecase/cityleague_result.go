package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type CityleagueResultInterface interface {
	FindByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
	) (*entity.CityleagueResult, error)

	FindByCityleagueScheduleId(
		ctx context.Context,
		leagueType uint,
		cityleagueScheduleId string,
	) ([]*entity.CityleagueResult, error)

	FindByDate(
		ctx context.Context,
		leagueType uint,
		date time.Time,
	) ([]*entity.CityleagueResult, error)

	FindByTerm(
		ctx context.Context,
		leagueType uint,
		fromDate time.Time,
		toDate time.Time,
	) ([]*entity.CityleagueResult, error)
}

type CityleagueResult struct {
	repository CityleagueResultInterface
}

func NewCityleagueResult(
	repository CityleagueResultInterface,
) *CityleagueResult {
	return &CityleagueResult{repository}
}

func (u *CityleagueResult) FindByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
) (*entity.CityleagueResult, error) {
	return u.repository.FindByOfficialEventId(ctx, officialEventId)
}

func (u *CityleagueResult) FindByCityleagueScheduleId(
	ctx context.Context,
	leagueType uint,
	cityleagueScheduleId string,
) ([]*entity.CityleagueResult, error) {
	return u.repository.FindByCityleagueScheduleId(ctx, leagueType, cityleagueScheduleId)
}

func (u *CityleagueResult) FindByDate(
	ctx context.Context,
	leagueType uint,
	date time.Time,
) ([]*entity.CityleagueResult, error) {
	return u.repository.FindByDate(ctx, leagueType, date)
}

func (u *CityleagueResult) FindByTerm(
	ctx context.Context,
	leagueType uint,
	fromDate time.Time,
	toDate time.Time,
) ([]*entity.CityleagueResult, error) {
	return u.repository.FindByTerm(ctx, leagueType, fromDate, toDate)
}
