package dto

import "time"

type UserPlayerCreateRequest struct {
	PlayerId string `json:"player_id"`
	// VerificationToken は webapp が所有権確認を終えたことを示す署名付きトークン。
	VerificationToken string `json:"verification_token"`
}

type UserPlayerResponse struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UserId      string    `json:"user_id"`
	PlayerId    string    `json:"player_id"`
	LockedUntil time.Time `json:"locked_until"`
}

type UserPlayerGetResponse struct {
	UserPlayerResponse
	// ChampionShipPoint / RankingDate はプレイヤーズクラブのランキング履歴が
	// 未登録の場合 nil になる(連携直後でまだ集計対象になっていない等)。
	ChampionShipPoint *int       `json:"champion_ship_point,omitempty"`
	RankingDate       *time.Time `json:"ranking_date,omitempty"`
}

type UserPlayerCreateResponse struct {
	UserPlayerResponse
}

// UserPlayerChallengeRequest は所有権確認で提示するアバターの払い出し要求。
// CurrentAvatarImage は webapp がプレイヤーズクラブから取得した現在のアバター画像URLで、
// 同じ画像を提示しても確認にならないため除外に使う。
type UserPlayerChallengeRequest struct {
	CurrentAvatarImage string `json:"current_avatar_image"`
}

type UserPlayerChallengeResponse struct {
	AvatarId       int    `json:"avatar_id"`
	AvatarTitle    string `json:"avatar_title"`
	AvatarImageURL string `json:"avatar_image_url"`
	AvatarDetail   string `json:"avatar_detail"`
}
