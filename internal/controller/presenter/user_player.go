package presenter

import (
	"fmt"
	"time"

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

func NewUserPlayerCityleagueResultsGetResponse(
	season string,
	playerCityleagueResults []*entity.PlayerCityleagueResult,
) *dto.UserPlayerCityleagueResultsGetResponse {
	results := []*dto.UserPlayerCityleagueResultResponse{}

	for _, playerCityleagueResult := range playerCityleagueResults {
		results = append(results, &dto.UserPlayerCityleagueResultResponse{
			OfficialEventId: playerCityleagueResult.OfficialEventId,
			LeagueType:      playerCityleagueResult.LeagueType,
			// 他のシティリーグ結果の応答と同じく、開催日は0時のローカル時刻に揃える。
			Date: time.Date(
				playerCityleagueResult.EventDate.Year(),
				playerCityleagueResult.EventDate.Month(),
				playerCityleagueResult.EventDate.Day(),
				0, 0, 0, 0, time.Local,
			),
			EventTitle:           playerCityleagueResult.EventTitle,
			ShopName:             playerCityleagueResult.ShopName,
			PrefectureName:       playerCityleagueResult.PrefectureName,
			EnvironmentTitle:     playerCityleagueResult.EnvironmentTitle,
			Rank:                 playerCityleagueResult.Rank,
			Point:                playerCityleagueResult.Point,
			DeckCode:             playerCityleagueResult.DeckCode,
			EventDetailResultURL: fmt.Sprintf("https://players.pokemon-card.com/event/detail/%d/result", playerCityleagueResult.OfficialEventId),
		})
	}

	return &dto.UserPlayerCityleagueResultsGetResponse{
		Season:  season,
		Count:   len(results),
		Results: results,
	}
}
