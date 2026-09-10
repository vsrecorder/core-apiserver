package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OpponentDeckUsageStatInterface interface {
	FindOpponentDeckUsageStat(
		ctx context.Context,
		userId string,
		period StatPeriod,
		deckId string,
		regulationId uint,
		// excludeDefaultMatches が true なら不戦勝/不戦敗(default_victory_flg /
		// default_defeat_flg)を集計から外す。不戦は対戦が行われていないため、
		// 相手デッキ名が記録されていても「そのデッキと当たった1戦」には数えない。
		excludeDefaultMatches bool,
	) (*entity.OpponentDeckUsageStat, error)
}
