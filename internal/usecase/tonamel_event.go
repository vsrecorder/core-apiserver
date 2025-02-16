package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type TonamelEventInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.TonamelEvent, error)
}

type TonamelEvent struct {
	repository repository.TonamelEventInterface
}

func NewTonamelEvent(
	repository repository.TonamelEventInterface,
) *TonamelEvent {
	return &TonamelEvent{repository}
}

func (i *TonamelEvent) FindById(
	ctx context.Context,
	id string,
) (*entity.TonamelEvent, error) {
	tonamelEvent, err := i.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return tonamelEvent, nil
}
