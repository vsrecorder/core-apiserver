package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewUserPlayerGetResponse(
	userPlayer *entity.UserPlayer,
	ranking *entity.PlayerRanking,
) *dto.UserPlayerGetResponse {
	res := &dto.UserPlayerGetResponse{
		UserPlayerResponse: dto.UserPlayerResponse{
			ID:          userPlayer.ID,
			CreatedAt:   userPlayer.CreatedAt,
			UserId:      userPlayer.UserId,
			PlayerId:    userPlayer.PlayerId,
			LockedUntil: userPlayer.LockedUntil(),
		},
	}

	if ranking != nil {
		res.ChampionShipPoint = &ranking.ChampionShipPoint
		res.RankingDate = &ranking.RankingDate
	}

	return res
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

func NewUserPlayerChallengeResponse(
	avatar *entity.PokemonAvatar,
) *dto.UserPlayerChallengeResponse {
	return &dto.UserPlayerChallengeResponse{
		AvatarId:       avatar.ID,
		AvatarTitle:    avatar.Title,
		AvatarImageURL: avatar.ImageURL,
		AvatarDetail:   avatar.Detail,
	}
}
