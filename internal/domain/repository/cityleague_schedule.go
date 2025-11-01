package repository

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
