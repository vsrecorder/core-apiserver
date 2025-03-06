package dto

import "time"

type UserResponse struct {
	ID          string `json:"id"`
	CreatedAt   time.Time
	DisplayName string `json:"display_name"`
	PhotoURL    string `json:"photo_url"`
}
type UserGetByIdResponse struct {
	UserResponse
}
