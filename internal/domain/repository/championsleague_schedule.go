package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type ChampionsleagueScheduleInterface interface {
	// Find は全ての大会を開催日の新しい順に返す。
	Find(
		ctx context.Context,
	) ([]*entity.ChampionsleagueSchedule, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.ChampionsleagueSchedule, error)
}
