package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type ChampionsleagueResultInterface interface {
	FindEvents(
		ctx context.Context,
	) ([]*entity.ChampionsleagueResultEvent, error)

	FindByChampionsleagueScheduleId(
		ctx context.Context,
		leagueType uint,
		championsleagueScheduleId string,
	) ([]*entity.ChampionsleagueResult, error)
}

type ChampionsleagueResult struct {
	repository ChampionsleagueResultInterface
}

func NewChampionsleagueResult(
	repository ChampionsleagueResultInterface,
) *ChampionsleagueResult {
	return &ChampionsleagueResult{repository}
}

func (u *ChampionsleagueResult) FindEvents(
	ctx context.Context,
) ([]*entity.ChampionsleagueResultEvent, error) {
	return u.repository.FindEvents(ctx)
}

func (u *ChampionsleagueResult) FindByChampionsleagueScheduleId(
	ctx context.Context,
	leagueType uint,
	championsleagueScheduleId string,
) ([]*entity.ChampionsleagueResult, error) {
	return u.repository.FindByChampionsleagueScheduleId(ctx, leagueType, championsleagueScheduleId)
}
