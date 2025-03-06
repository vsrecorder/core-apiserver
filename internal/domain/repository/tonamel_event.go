package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type TonamelEventInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.TonamelEvent, error)
}
