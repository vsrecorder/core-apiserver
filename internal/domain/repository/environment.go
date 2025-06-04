package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type EnvironmentInterface interface {
	Find(
		ctx context.Context,
	) ([]*entity.Environment, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.Environment, error)

	FindByDate(
		ctx context.Context,
		date time.Time,
	) (*entity.Environment, error)

	FindByTerm(
		ctx context.Context,
		fromDate time.Time,
		toDate time.Time,
	) ([]*entity.Environment, error)
}
