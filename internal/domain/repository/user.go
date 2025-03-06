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
}
