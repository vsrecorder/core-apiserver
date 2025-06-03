package dto

import "time"

type UserRequest struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

type UserCreateRequest struct {
	UserRequest
}

type UserUpdateRequest struct {
	UserRequest
}

type UserResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	ImageURL  string    `json:"image_url"`
}

type UserGetByIdResponse struct {
	UserResponse
}

type UserCreateResponse struct {
	UserResponse
}

type UserUpdateResponse struct {
	UserResponse
}
