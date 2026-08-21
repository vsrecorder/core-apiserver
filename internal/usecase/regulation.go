package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type RegulationInterface interface {
	Find(
		ctx context.Context,
	) ([]*entity.Regulation, error)
}

type Regulation struct {
	repository RegulationInterface
}

func NewRegulation(
	repository RegulationInterface,
) *Regulation {
	return &Regulation{repository}
}

func (u *Regulation) Find(
	ctx context.Context,
) ([]*entity.Regulation, error) {
	return u.repository.Find(ctx)
}
