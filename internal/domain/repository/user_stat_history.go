package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserStatHistoryInterface interface {
	// excludeDefaultMatches が true なら不戦勝/不戦敗(default_victory_flg /
	// default_defeat_flg)を月ごとの集計から外す。
	FindUserStatHistory(ctx context.Context, userId string, fromDate time.Time, toDate time.Time, deckId string, regulationId uint, excludeDefaultMatches bool) ([]*entity.UserStatMonthly, error)
}
