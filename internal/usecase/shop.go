package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// ShopSearchDefaultLimit は店舗検索で limit の指定が無いときに返す件数。
	// Myジムを選ぶための検索なので、絞り込みを促す程度の件数に留める。
	ShopSearchDefaultLimit = 50

	// ShopSearchMaxLimit は店舗検索で返せる件数の上限。
	// 店舗マスタは数千件あり、上限が無いと全件をJSONにして返せてしまう。
	ShopSearchMaxLimit = 100
)

type ShopInterface interface {
	Find(
		ctx context.Context,
		keyword string,
		limit int,
	) ([]*entity.Shop, error)
}

type Shop struct {
	repository repository.ShopInterface
}

func NewShop(
	repository repository.ShopInterface,
) ShopInterface {
	return &Shop{repository}
}

func (u *Shop) Find(
	ctx context.Context,
	keyword string,
	limit int,
) ([]*entity.Shop, error) {
	if limit <= 0 {
		limit = ShopSearchDefaultLimit
	} else if limit > ShopSearchMaxLimit {
		limit = ShopSearchMaxLimit
	}

	shops, err := u.repository.Find(ctx, keyword, limit)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return shops, nil
}
