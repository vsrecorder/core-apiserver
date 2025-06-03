package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.User, error)

	Save(
		ctx context.Context,
		entity *entity.User,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
