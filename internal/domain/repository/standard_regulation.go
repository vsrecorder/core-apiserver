package repository

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
