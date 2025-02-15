package dto

type UserResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	PhotoURL    string `json:"photo_url"`
}
type UserGetByIdResponse struct {
	UserResponse
}
