package dto

import "time"

type UserPlayerCreateRequest struct {
	PlayerId string `json:"player_id"`
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
}

type UserPlayerCreateResponse struct {
	UserPlayerResponse
}

// UserPlayerCityleagueResultResponse は連携済みプレイヤーIDの入賞1件。
// トレーナー情報ページでは入賞1件を1枚のカードとして並べるため、
// CityleagueResultGetResponse のようなイベント単位の入れ子にはしない。
type UserPlayerCityleagueResultResponse struct {
	OfficialEventId      uint      `json:"official_event_id"`
	LeagueType           uint      `json:"league_type"`
	Date                 time.Time `json:"date"`
	EventTitle           string    `json:"event_title"`
	ShopName             string    `json:"shop_name"`
	PrefectureName       string    `json:"prefecture_name"`
	EnvironmentTitle     string    `json:"environment_title"`
	Rank                 uint      `json:"rank"`
	Point                uint      `json:"point"`
	DeckCode             string    `json:"deck_code"`
	EventDetailResultURL string    `json:"event_detail_result_url"`
}

type UserPlayerCityleagueResultsGetResponse struct {
	Season  string                                `json:"season"`
	Count   int                                   `json:"count"`
	Results []*UserPlayerCityleagueResultResponse `json:"results"`
}
