package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type GameInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Game, error)

	FindByMatchId(
		ctxt context.Context,
		matchId string,
	) ([]*entity.Game, error)
}

type Game struct {
	repository repository.GameInterface
}

func NewGame(
	repository repository.GameInterface,
) GameInterface {
	return &Game{repository}
}

func (u *Game) FindById(
	ctx context.Context,
	id string,
) (*entity.Game, error) {
	game, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return game, nil
}

func (u *Game) FindByMatchId(
	ctx context.Context,
	matchId string,
) ([]*entity.Game, error) {
	games, err := u.repository.FindByMatchId(ctx, matchId)

	if err != nil {
		return nil, err
	}

	return games, nil
}
