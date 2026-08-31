package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OfficialEventInterface interface {
	Find(
		ctx context.Context,
		typeId uint,
		leagueType uint,
		startDate time.Time,
		endDate time.Time,
	) ([]*entity.OfficialEvent, error)

	FindById(
		ctx context.Context,
		id uint,
	) (*entity.OfficialEvent, error)

	// FindByShopIds は指定した店舗で開催される公式イベントを、期間内について
	// 開催日時の昇順で返す。Myジム(user_gyms)のイベント一覧に使う。
	//
	// shopIds が空の場合は空スライスを返す(Myジム未登録のユーザで全件を引かないため)。
	// 種別(type_id)では絞らない。Myジムのパネルは「その店で何があるか」を
	// まとめて見せ、種別の出し分けは表示側が行う。
	FindByShopIds(
		ctx context.Context,
		shopIds []uint,
		startDate time.Time,
		endDate time.Time,
	) ([]*entity.OfficialEvent, error)
}
