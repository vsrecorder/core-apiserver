package usecase

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

type ChampionshipSeries struct {
	repository ChampionshipSeriesInterface
}

func NewChampionshipSeries(
	repository ChampionshipSeriesInterface,
) *ChampionshipSeries {
	return &ChampionshipSeries{repository}
}

func (u *ChampionshipSeries) Find(
	ctx context.Context,
) ([]*entity.ChampionshipSeries, error) {
	return u.repository.Find(ctx)
}

func (u *ChampionshipSeries) FindById(
	ctx context.Context,
	id string,
) (*entity.ChampionshipSeries, error) {
	return u.repository.FindById(ctx, id)
}

func (u *ChampionshipSeries) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.ChampionshipSeries, error) {
	return u.repository.FindByDate(ctx, date)
}
