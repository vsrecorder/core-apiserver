package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// TonamelEventStoreInterface は Tonamel の大会情報を保存・参照する永続化層。
//
// TonamelEventInterface が tonamel.com からの取得(HTTP)を担うのに対し、こちらは
// 取得結果をDBにためて再利用するためのもの。大会情報はほぼ不変で全ユーザー共通なので、
// 記録作成時に1度だけ取得して Save し、カレンダー等の参照は FindByIds で1クエリにまとめる。
type TonamelEventStoreInterface interface {
	// FindByIds は指定したID群に対応する大会情報をまとめて返す。
	// 見つからないIDは結果に含めない(存在しないことをエラーにはしない)。
	FindByIds(
		ctx context.Context,
		ids []string,
	) ([]*entity.TonamelEvent, error)

	// Save は大会情報を保存する。同じIDが既にあれば上書きする(upsert)。
	Save(
		ctx context.Context,
		tonamelEvent *entity.TonamelEvent,
	) error
}
