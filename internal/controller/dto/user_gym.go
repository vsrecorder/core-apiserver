package dto

import "time"

type UserGymResponse struct {
	Shop *ShopResponse `json:"shop"`
	// CreatedAt はMyジムに登録した日時。一覧は登録順(古い順)に並ぶ。
	CreatedAt time.Time `json:"created_at"`
}

type UserGymGetResponse struct {
	// Limit は1ユーザが登録できる上限(usecase.MaxUserGymsPerUser)。
	// 「あと何枠空いているか」をクライアント側で数字を持たずに出せるようにする。
	Limit    int                `json:"limit"`
	Count    int                `json:"count"`
	UserGyms []*UserGymResponse `json:"user_gyms"`
}

type UserGymCreateRequest struct {
	ShopId uint `json:"shop_id"`
}

type UserGymCreateResponse struct {
	UserGymResponse
}

// UserGymOfficialEventGetResponse はMyジムと、その店舗の公式イベントを併せて返す。
// パネルは「登録している店」と「そこでの予定」を同時に描くため1度の取得で揃える。
type UserGymOfficialEventGetResponse struct {
	StartDate      time.Time                `json:"start_date"`
	EndDate        time.Time                `json:"end_date"`
	Limit          int                      `json:"limit"`
	UserGyms       []*UserGymResponse       `json:"user_gyms"`
	Count          int                      `json:"count"`
	OfficialEvents []*OfficialEventResponse `json:"official_events"`
}
