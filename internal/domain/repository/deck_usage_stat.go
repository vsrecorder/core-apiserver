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

	// FindDeckUsageStatBySprites は FindDeckUsageStat と同じ集計を、同じデッキでも対戦で使った
	// スプライトの組み合わせ(記録が使ったバージョンのスプライト)ごとに行を分けて返す。
	// 同じ DeckId の行が複数返りうる。集計対象外・不戦の件数(デッキ単位)は、そのデッキの
	// 先頭の行にだけ入る。使用率を並べて見せる画面向けで、デッキIDで引く用途には使わない。
	FindDeckUsageStatBySprites(
		ctx context.Context,
		userId string,
		period StatPeriod,
		regulationId uint,
		excludeDefaultMatches bool,
	) (*entity.DeckUsageStat, error)

	// FindDeckCodeUsageStat は userId のデッキ deckId の対戦成績をバージョン(デッキコード)ごとに集計する。
	// 期間は区切らない(全期間)。集計対象外の記録(ignore_stats_flg)は FindDeckUsageStat と同じく除く。
	FindDeckCodeUsageStat(
		ctx context.Context,
		userId string,
		deckId string,
		excludeDefaultMatches bool,
	) (*entity.DeckCodeUsageStat, error)
}
