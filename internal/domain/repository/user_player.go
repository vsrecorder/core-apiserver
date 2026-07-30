package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserPlayerInterface interface {
	FindByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserPlayer, error)

	Save(
		ctx context.Context,
		entity *entity.UserPlayer,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
