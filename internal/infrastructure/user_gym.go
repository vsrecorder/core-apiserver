package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// userGymView は user_gyms と shops(+prefectures)を結合した1行を受ける。
//
// model.Shop を埋め込まずフィールドを並べているのは、GORM が匿名フィールドを
// 平坦化せず別テーブルの関連として扱うため(Scan の結果が埋まらない)。
type userGymView struct {
	ID             uint
	Name           string
	ZipCode        string
	PrefectureId   uint
	PrefectureName string
	Address        string
	Tel            string
	BusinessHours  string
	URL            string
	CreatedAt      time.Time
}

type UserGym struct {
	db *gorm.DB
}

func NewUserGym(db *gorm.DB) repository.UserGymInterface {
	return &UserGym{db}
}

func (i *UserGym) FindByUserId(
	ctx context.Context,
	uid string,
) ([]*entity.UserGymView, error) {
	db := dbFromContext(ctx, i.db)

	var views []*userGymView

	// 内部結合。user_gyms.shop_id には shops へのFKがあり、取り込みバッチ
	// (import-officialevent-bat)も店舗を Save するだけで消さないため、
	// 参照先を欠いた行は通常できない。それでも外部結合にしないのは、万一
	// 参照先を失った行ができたときに、店舗名の無い枠を一覧へ出さないため。
	if tx := db.Table(
		"user_gyms",
	).Select(
		shopSelect+", user_gyms.created_at AS created_at",
	).Joins(
		"JOIN shops ON shops.id = user_gyms.shop_id",
	).Joins(
		"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
	).Where(
		"user_gyms.user_id = ?", uid,
	).Order(
		"user_gyms.created_at ASC, user_gyms.shop_id ASC",
	).Scan(&views); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := make([]*entity.UserGymView, 0, len(views))
	for _, view := range views {
		ret = append(ret, entity.NewUserGymView(
			entity.NewShop(
				view.ID,
				view.Name,
				view.ZipCode,
				view.PrefectureId,
				view.PrefectureName,
				view.Address,
				view.Tel,
				view.BusinessHours,
				view.URL,
			),
			view.CreatedAt,
		))
	}

	return ret, nil
}

// LockByUserId はユーザ単位のアドバイザリロックを取る。
//
// pg_advisory_xact_lock はトランザクションの終了で自動解放されるため、
// 明示的な解放は要らない(途中でエラーになってもロックが残らない)。
func (i *UserGym) LockByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Exec(
		"SELECT pg_advisory_xact_lock(hashtext(?))", uid,
	); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}

func (i *UserGym) Create(
	ctx context.Context,
	entity *entity.UserGym,
) error {
	db := dbFromContext(ctx, i.db)

	userGym := model.NewUserGym(
		entity.UserId,
		entity.ShopId,
		entity.CreatedAt,
	)

	// 同じ店舗への同時登録は主キー違反になる。そのまま返すと500になってしまうため、
	// 「既に登録済み」として扱えるよう変換する(呼び出し側が409を返せる)。
	if tx := db.Create(userGym); tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapUniqueViolation(tx.Error)
	}

	return nil
}

func (i *UserGym) Delete(
	ctx context.Context,
	uid string,
	shopId uint,
) error {
	if tx := dbFromContext(ctx, i.db).Where(
		"user_id = ? AND shop_id = ?", uid, shopId,
	).Delete(&model.UserGym{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}

// DeleteByUserId は退会時に、そのユーザのMyジムを行ごと削除する。
// このテーブルは論理削除を持たないため物理削除する。
func (i *UserGym) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Where(
		"user_id = ?", uid,
	).Delete(&model.UserGym{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}
