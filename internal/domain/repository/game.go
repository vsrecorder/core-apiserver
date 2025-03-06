package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type GameInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Game, error)

	FindByMatchId(
		ctx context.Context,
		matchId string,
	) ([]*entity.Game, error)

	/*
		Save(
			ctx context.Context,
			entity *entity.Game,
		) error

		Delete(
			ctx context.Context,
			id string,
		) error
	*/
}
