package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type TonamelEventInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.TonamelEvent, error)
}

type TonamelEvent struct {
	// repository は tonamel.com から大会情報を取得する(HTTP)。
	repository repository.TonamelEventInterface
	// store は取得済みの大会情報(tonamel_events テーブル)。
	store repository.TonamelEventStoreInterface
}

func NewTonamelEvent(
	repository repository.TonamelEventInterface,
	store repository.TonamelEventStoreInterface,
) TonamelEventInterface {
	return &TonamelEvent{repository, store}
}

/*
 * FindById は大会情報を返す。保存済み(tonamel_events)ならそれを返し、無いときだけ
 * tonamel.com へ取りに行って保存する。
 *
 * この経路は未認証で叩けるため、毎回外部サイトへ取りに行く作りだと、大量に叩かれるだけで
 * 外向きの接続が滞留し(1件あたり最大でタイムアウトの10秒)、tonamel.com 側から遮断されれば
 * 記録作成の連携まで止まる。記録作成時(Record.persistTonamelEvent)が同じテーブルへ保存して
 * いるので、既知のIDは外部通信なしで返せる。
 *
 * 保存の失敗は取得結果の返却を妨げない(次回また取りに行くだけ)。保存済みかの確認に失敗した
 * 場合も、取得へ進んで応答は返す。
 */
func (u *TonamelEvent) FindById(
	ctx context.Context,
	id string,
) (*entity.TonamelEvent, error) {
	stored, err := u.store.FindByIds(ctx, []string{id})
	if err != nil {
		logWarn(ctx, err)
	} else if len(stored) > 0 {
		return stored[0], nil
	}

	tonamelEvent, err := u.repository.FindById(ctx, id)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	if err := u.store.Save(ctx, tonamelEvent); err != nil {
		logWarn(ctx, err)
	}

	return tonamelEvent, nil
}
