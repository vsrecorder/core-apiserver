package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type OfficialEventInterface interface {
	Find(
		ctx context.Context,
		typeId uint,
		leagueType uint,
		startDate time.Time,
		endDate time.Time,
	) ([]*entity.OfficialEvent, error)

	FindById(
		ctx context.Context,
		id uint,
	) (*entity.OfficialEvent, error)
}

type OfficialEvent struct {
	repository repository.OfficialEventInterface
}

func NewOfficialEvent(
	repository repository.OfficialEventInterface,
) OfficialEventInterface {
	return &OfficialEvent{repository}
}

func (u *OfficialEvent) Find(
	ctx context.Context,
	typeId uint,
	leagueType uint,
	startDate time.Time,
	endDate time.Time,
) ([]*entity.OfficialEvent, error) {
	officialEvents, err := u.repository.Find(ctx, typeId, leagueType, startDate, endDate)

	if err != nil {
		return nil, err
	}

	return officialEvents, nil
}

func (u *OfficialEvent) FindById(
	ctx context.Context,
	id uint,
) (*entity.OfficialEvent, error) {
	officialEvent, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return officialEvent, nil
}
