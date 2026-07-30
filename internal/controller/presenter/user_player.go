package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewUserPlayerGetResponse(
	userPlayer *entity.UserPlayer,
) *dto.UserPlayerGetResponse {
	return &dto.UserPlayerGetResponse{
		UserPlayerResponse: dto.UserPlayerResponse{
			ID:          userPlayer.ID,
			CreatedAt:   userPlayer.CreatedAt,
			UserId:      userPlayer.UserId,
			PlayerId:    userPlayer.PlayerId,
			LockedUntil: userPlayer.LockedUntil(),
		},
	}
}

func NewUserPlayerCreateResponse(
	userPlayer *entity.UserPlayer,
) *dto.UserPlayerCreateResponse {
	return &dto.UserPlayerCreateResponse{
		UserPlayerResponse: dto.UserPlayerResponse{
			ID:          userPlayer.ID,
			CreatedAt:   userPlayer.CreatedAt,
			UserId:      userPlayer.UserId,
			PlayerId:    userPlayer.PlayerId,
			LockedUntil: userPlayer.LockedUntil(),
		},
	}
}
