package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type RegulationInterface interface {
	Find(
		ctx context.Context,
	) ([]*entity.Regulation, error)
}
