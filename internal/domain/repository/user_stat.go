package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserStatInterface interface {
	FindUserStat(
		ctx context.Context,
		userId string,
		period StatPeriod,
		regulationId uint,
		// excludeDefaultMatches が true なら不戦勝/不戦敗(default_victory_flg /
		// default_defeat_flg)を対戦の集計から外す。記録数・イベント数は
		// 「記録した回数」であって対戦の有無とは無関係なので、この指定でも変わらない。
		excludeDefaultMatches bool,
	) (*entity.UserStat, error)
}
