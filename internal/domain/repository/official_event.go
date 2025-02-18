package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OfficialEventInterface interface {
	Find(
		ctx context.Context,
		typeId uint,
		leagueType uint,
		startDate time.Time,
		endDate time.Time,
	) ([]*entity.OfficialEvent, error)

	FindById(
		ctx context.Context,
		id uint,
	) (*entity.OfficialEvent, error)
}
