package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type StandardRegulationInterface interface {
	Find(
		ctx context.Context,
	) ([]*entity.StandardRegulation, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.StandardRegulation, error)

	FindByDate(
		ctx context.Context,
		date time.Time,
	) (*entity.StandardRegulation, error)
}

type StandardRegulation struct {
	repository StandardRegulationInterface
}

func NewStandardRegulation(
	repository StandardRegulationInterface,
) *StandardRegulation {
	return &StandardRegulation{repository}
}

func (u *StandardRegulation) Find(
	ctx context.Context,
) ([]*entity.StandardRegulation, error) {
	return u.repository.Find(ctx)
}

func (u *StandardRegulation) FindById(
	ctx context.Context,
	id string,
) (*entity.StandardRegulation, error) {
	return u.repository.FindById(ctx, id)
}

func (u *StandardRegulation) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.StandardRegulation, error) {
	return u.repository.FindByDate(ctx, date)
}
