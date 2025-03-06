package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.User, error)
}

type User struct {
	repository repository.UserInterface
}

func NewUser(
	repository repository.UserInterface,
) UserInterface {
	return &User{repository}
}

func (u *User) FindById(
	ctx context.Context,
	id string,
) (*entity.User, error) {
	user, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return user, nil
}
