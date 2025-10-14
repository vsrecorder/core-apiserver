package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

var (
	ErrAlreadyExists = errors.New("already exists")
)

type UserCreateParam struct {
	ID       string
	Name     string
	ImageURL string
}

type UserUpdateParam struct {
	Name     string
	ImageURL string
}

func NewUserCreateParam(
	id string,
	name string,
	imageURL string,
) *UserCreateParam {
	return &UserCreateParam{
		ID:       id,
		Name:     name,
		ImageURL: imageURL,
	}
}

func NewUserUpdateParam(
	name string,
	imageURL string,
) *UserUpdateParam {
	return &UserUpdateParam{
		Name:     name,
		ImageURL: imageURL,
	}
}

type UserInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.User, error)

	Create(
		ctx context.Context,
		param *UserCreateParam,
	) (*entity.User, error)

	Update(
		ctx context.Context,
		id string,
		param *UserUpdateParam,
	) (*entity.User, error)

	Delete(
		ctx context.Context,
		id string,
	) error
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

func (u *User) Create(
	ctx context.Context,
	param *UserCreateParam,
) (*entity.User, error) {
	createdAt := time.Now().Local()

	user := entity.NewUser(
		param.ID,
		createdAt,
		param.Name,
		param.ImageURL,
	)

	_, err := u.repository.FindById(ctx, user.ID)
	if err == nil {
		return nil, ErrAlreadyExists
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err := u.repository.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Update(
	ctx context.Context,
	id string,
	param *UserUpdateParam,
) (*entity.User, error) {
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	user := entity.NewUser(
		id,
		ret.CreatedAt,
		param.Name,
		param.ImageURL,
	)

	if err := u.repository.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Delete(
	ctx context.Context,
	id string,
) error {
	err := u.repository.Delete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
