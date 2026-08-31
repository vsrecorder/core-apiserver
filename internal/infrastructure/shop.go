package infrastructure

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// shopSelect は shops と prefectures を結合して店舗を引くときの選択列。
// 都道府県名まで揃えないと一覧で「どこの店か」が分からないため、常に結合して取る。
const shopSelect = `
	shops.id AS id,
	shops.name AS name,
	shops.zip_code AS zip_code,
	prefectures.id AS prefecture_id,
	prefectures.name AS prefecture_name,
	shops.address AS address,
	shops.tel AS tel,
	shops.business_hours AS business_hours,
	shops.url AS url
`

// escapeLikePattern は LIKE のワイルドカード(% _)とエスケープ文字自身を無効化する。
//
// 利用者が入力したキーワードをそのまま LIKE に渡すと、"%" 一文字で全件一致になり
// limit を付けていても店舗マスタ全体を舐めるクエリになる。値のバインド自体は
// GORM が行うためSQLインジェクションにはならないが、パターンとしての意味は消す。
func escapeLikePattern(keyword string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

	return r.Replace(keyword)
}

type Shop struct {
	db *gorm.DB
}

func NewShop(db *gorm.DB) repository.ShopInterface {
	return &Shop{db}
}

func (i *Shop) Find(
	ctx context.Context,
	keyword string,
	limit int,
) ([]*entity.Shop, error) {
	db := dbFromContext(ctx, i.db)

	pattern := "%" + escapeLikePattern(keyword) + "%"

	var shops []*model.Shop
	if tx := db.Table(
		"shops",
	).Select(
		shopSelect,
	).Joins(
		"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
	).Where(
		// id = 0 は「株式会社ポケモン」で、店舗ではなく大型大会の主催者として入っている。
		// Myジムとして選べても意味がないため検索結果から除く。
		"shops.id > 0",
	).Where(
		"(shops.name ILIKE ? OR shops.address ILIKE ?)", pattern, pattern,
	).Order(
		// 全国チェーンを引いたときに地域ごとにまとまって出るよう都道府県から並べる
		"shops.prefecture_id ASC, shops.name ASC, shops.id ASC",
	).Limit(
		limit,
	).Scan(&shops); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := make([]*entity.Shop, 0, len(shops))
	for _, shop := range shops {
		ret = append(ret, newShopEntity(shop))
	}

	return ret, nil
}

func (i *Shop) FindById(
	ctx context.Context,
	id uint,
) (*entity.Shop, error) {
	db := dbFromContext(ctx, i.db)

	var shop model.Shop
	if tx := db.Table(
		"shops",
	).Select(
		shopSelect,
	).Joins(
		"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
	).Where(
		"shops.id = ?", id,
	).First(&shop); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	return newShopEntity(&shop), nil
}

func newShopEntity(shop *model.Shop) *entity.Shop {
	return entity.NewShop(
		shop.ID,
		shop.Name,
		shop.ZipCode,
		shop.PrefectureId,
		shop.PrefectureName,
		shop.Address,
		shop.Tel,
		shop.BusinessHours,
		shop.URL,
	)
}
