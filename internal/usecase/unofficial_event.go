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

	Update(
		ctx context.Context,
		id string,
		param *UnofficialEventParam,
	) (*entity.UnofficialEvent, error)

	Delete(
		ctx context.Context,
		id string,
	) error
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

func (u *UnofficialEvent) Update(
	ctx context.Context,
	id string,
	param *UnofficialEventParam,
) (*entity.UnofficialEvent, error) {
	// 指定されたidのUnofficialEventが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	unofficialEvent := entity.NewUnofficialEvent(
		id,
		param.userId,
		param.title,
		param.date,
	)
	// 作成日時は更新対象ではないため、保存済みの値を引き継ぐ
	unofficialEvent.CreatedAt = ret.CreatedAt

	if err := u.repository.Save(ctx, unofficialEvent); err != nil {
		return nil, err
	}

	return unofficialEvent, nil
}

func (u *UnofficialEvent) Delete(
	ctx context.Context,
	id string,
) error {
	// 指定されたidのUnofficialEventが存在するか確認
	if _, err := u.repository.FindById(ctx, id); err != nil {
		return err
	}

	return u.repository.Delete(ctx, id)
}
