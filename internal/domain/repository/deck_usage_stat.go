package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type DeckUsageStatInterface interface {
	FindDeckUsageStat(
		ctx context.Context,
		userId string,
		period StatPeriod,
		regulationId uint,
		// excludeDefaultMatches が true なら不戦勝/不戦敗(default_victory_flg /
		// default_defeat_flg)をデッキごとの集計から外す。対戦数・勝敗・使用率の分母の
		// すべてから外れるため、そのデッキで不戦しか記録が無い場合は結果に現れなくなる。
		excludeDefaultMatches bool,
	) (*entity.DeckUsageStat, error)
}
