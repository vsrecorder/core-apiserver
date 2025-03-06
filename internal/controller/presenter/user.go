package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewUserGetByIdResponse(
	user *entity.User,
) *dto.UserGetByIdResponse {
	return &dto.UserGetByIdResponse{
		UserResponse: dto.UserResponse{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			DisplayName: user.DisplayName,
			PhotoURL:    user.PhotoURL,
		},
	}
}
