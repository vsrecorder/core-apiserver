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

	ExistsActiveByPlayerId(
		ctx context.Context,
		playerId string,
	) (bool, error)

	Save(
		ctx context.Context,
		entity *entity.UserPlayer,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
