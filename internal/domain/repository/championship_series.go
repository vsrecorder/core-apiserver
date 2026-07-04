package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type ChampionshipSeriesInterface interface {
	Find(
		ctx context.Context,
	) ([]*entity.ChampionshipSeries, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.ChampionshipSeries, error)

	FindByDate(
		ctx context.Context,
		date time.Time,
	) (*entity.ChampionshipSeries, error)
}
