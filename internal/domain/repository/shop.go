package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type ShopInterface interface {
	// Find は keyword(店舗名・住所への部分一致。大文字小文字を無視)で店舗を検索する。
	// 店舗マスタは公式サイト由来で数千件あり全件返すには多いため、limit で必ず打ち切る。
	//
	// 並び順は都道府県・店舗名の昇順。「BOOKOFF」のような全国チェーンを引いたときに
	// 地域ごとにまとまって出るようにするためで、同じ条件なら常に同じ結果になる。
	Find(
		ctx context.Context,
		keyword string,
		limit int,
	) ([]*entity.Shop, error)

	// FindById は店舗を1件返す。存在しない場合は apperror.ErrRecordNotFound を返す。
	// Myジムに登録しようとしている店舗が実在するかの確認に使う。
	FindById(
		ctx context.Context,
		id uint,
	) (*entity.Shop, error)
}
