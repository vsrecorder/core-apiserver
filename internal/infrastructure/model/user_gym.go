package model

import "time"

// UserGym は user_gyms テーブルに対応する。
//
// shops の列ではなく別テーブルなのは、1ユーザが複数の店舗を登録するため。
// 解除は行の削除で表すため、論理削除(DeletedAt)は持たない(UserFavoriteDeck と同じ)。
type UserGym struct {
	UserId    string `gorm:"primaryKey"`
	ShopId    uint   `gorm:"primaryKey"`
	CreatedAt time.Time
}

func NewUserGym(
	userId string,
	shopId uint,
	createdAt time.Time,
) *UserGym {
	return &UserGym{
		UserId:    userId,
		ShopId:    shopId,
		CreatedAt: createdAt,
	}
}
