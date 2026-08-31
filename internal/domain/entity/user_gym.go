package entity

import "time"

// UserGym は「あるユーザがある店舗をMyジムとして登録している」ことを表す。
// CreatedAt は登録した日時で、一覧の並び順(古い順)に使う。
type UserGym struct {
	UserId    string
	ShopId    uint
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

// UserGymView はMyジム1件を、表示に必要な店舗情報まで揃えて表したもの。
// 一覧は必ず店舗名と所在地を伴って出すため、user_gyms と shops を結合して組み立てる。
type UserGymView struct {
	Shop      *Shop
	CreatedAt time.Time
}

func NewUserGymView(
	shop *Shop,
	createdAt time.Time,
) *UserGymView {
	return &UserGymView{
		Shop:      shop,
		CreatedAt: createdAt,
	}
}
