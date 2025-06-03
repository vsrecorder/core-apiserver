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
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			Name:      user.Name,
			ImageURL:  user.ImageURL,
		},
	}
}

func NewUserCreateResponse(
	user *entity.User,
) *dto.UserCreateResponse {
	return &dto.UserCreateResponse{
		UserResponse: dto.UserResponse{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			Name:      user.Name,
			ImageURL:  user.ImageURL,
		},
	}
}

func NewUserUpdateResponse(
	user *entity.User,
) *dto.UserUpdateResponse {
	return &dto.UserUpdateResponse{
		UserResponse: dto.UserResponse{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			Name:      user.Name,
			ImageURL:  user.ImageURL,
		},
	}
}
