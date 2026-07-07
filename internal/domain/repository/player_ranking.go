package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type PlayerRankingInterface interface {
	// FindLatestByPlayerId は player_id に紐づくランキング履歴のうち、
	// ranking_date が最新の1件を返す。存在しない場合は apperror.ErrRecordNotFound。
	FindLatestByPlayerId(
		ctx context.Context,
		playerId string,
	) (*entity.PlayerRanking, error)
}
