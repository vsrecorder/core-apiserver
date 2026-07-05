package dto

import "time"

type UserPlayerCreateRequest struct {
	PlayerId       string `json:"player_id"`
	ChallengeToken string `json:"challenge_token"`
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

type UserPlayerOwnershipChallengeResponse struct {
	Token          string    `json:"token"`
	AvatarId       int       `json:"avatar_id"`
	AvatarTitle    string    `json:"avatar_title"`
	AvatarImageURL string    `json:"avatar_image_url"`
	AvatarDetail   string    `json:"avatar_detail"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type UserPlayerVerifyResponse struct {
	PlayerId      string                               `json:"player_id"`
	Nickname      string                               `json:"nickname"`
	AvatarImage   string                               `json:"avatar_image"`
	CurrentLeague string                               `json:"current_league"`
	Prefecture    string                               `json:"prefecture"`
	Challenge     UserPlayerOwnershipChallengeResponse `json:"challenge"`
}
