package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UnofficialEventParam struct {
	userId string
	title  string
	date   time.Time
}

func NewUnofficialEventParam(
	userId string,
	title string,
	date time.Time,
) *UnofficialEventParam {
	return &UnofficialEventParam{
		userId: userId,
		title:  title,
		date:   date,
	}
}

type UnofficialEventInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.UnofficialEvent, error)

	Create(
		ctx context.Context,
		param *UnofficialEventParam,
	) (*entity.UnofficialEvent, error)
}

type UnofficialEvent struct {
	repository repository.UnofficialEventInterface
}

func NewUnofficialEvent(
	repository repository.UnofficialEventInterface,
) UnofficialEventInterface {
	return &UnofficialEvent{repository}
}

func (u *UnofficialEvent) FindById(
	ctx context.Context,
	id string,
) (*entity.UnofficialEvent, error) {
	unofficialEvent, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return unofficialEvent, nil
}

func (u *UnofficialEvent) Create(
	ctx context.Context,
	param *UnofficialEventParam,
) (*entity.UnofficialEvent, error) {
	id, err := generateId()
	if err != nil {
		return nil, err
	}

	unofficialEvent := entity.NewUnofficialEvent(
		id,
		param.userId,
		param.title,
		param.date,
	)

	if err := u.repository.Save(ctx, unofficialEvent); err != nil {
		return nil, err
	}

	return unofficialEvent, nil
}
