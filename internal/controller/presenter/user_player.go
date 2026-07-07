package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
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

func NewUserPlayerVerifyResponse(
	verification *usecase.UserPlayerVerification,
) *dto.UserPlayerVerifyResponse {
	return &dto.UserPlayerVerifyResponse{
		PlayerId:      verification.Account.PlayerId,
		Nickname:      verification.Account.Nickname,
		AvatarImage:   verification.Account.AvatarImage,
		CurrentLeague: verification.Account.CurrentLeague,
		Prefecture:    verification.Account.Prefecture,
		Challenge: dto.UserPlayerOwnershipChallengeResponse{
			Token:          verification.Challenge.Token,
			AvatarId:       verification.Challenge.ChallengeAvatarID,
			AvatarTitle:    verification.Challenge.ChallengeAvatarTitle,
			AvatarImageURL: verification.Challenge.ChallengeAvatarImageURL,
			AvatarDetail:   verification.Challenge.ChallengeAvatarDetail,
			ExpiresAt:      verification.Challenge.ExpiresAt,
		},
	}
}
