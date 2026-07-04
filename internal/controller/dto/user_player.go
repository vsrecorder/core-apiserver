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

type UserPlayerVerifyRequest struct {
	PlayerId string `json:"player_id"`
}

type UserPlayerVerifyResponse struct {
	PlayerId      string `json:"player_id"`
	Nickname      string `json:"nickname"`
	AvatarImage   string `json:"avatar_image"`
	CurrentLeague string `json:"current_league"`
	Prefecture    string `json:"prefecture"`
}
