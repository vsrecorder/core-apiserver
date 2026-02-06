package repository

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
		cityleagueIdScheduleId string,
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
