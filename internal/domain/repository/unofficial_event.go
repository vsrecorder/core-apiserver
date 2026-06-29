package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UnofficialEventInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.UnofficialEvent, error)

	Save(
		ctx context.Context,
		entity *entity.UnofficialEvent,
	) error
}
