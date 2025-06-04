package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
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

type Environment struct {
	repository repository.EnvironmentInterface
}

func NewEnvironment(
	repository repository.EnvironmentInterface,
) EnvironmentInterface {
	return &Environment{repository}
}

func (u *Environment) Find(
	ctx context.Context,
) ([]*entity.Environment, error) {
	environments, err := u.repository.Find(ctx)

	if err != nil {
		return nil, err
	}

	return environments, nil
}

func (u *Environment) FindById(
	ctx context.Context,
	id string,
) (*entity.Environment, error) {
	environment, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return environment, nil
}

func (u *Environment) FindByDate(
	ctx context.Context,
	date time.Time,
) (*entity.Environment, error) {
	environment, err := u.repository.FindByDate(ctx, date)

	if err != nil {
		return nil, err
	}

	return environment, nil
}

func (u *Environment) FindByTerm(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) ([]*entity.Environment, error) {
	environments, err := u.repository.FindByTerm(ctx, fromDate, toDate)

	if err != nil {
		return nil, err
	}

	return environments, nil
}
