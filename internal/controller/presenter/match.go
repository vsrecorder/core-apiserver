package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewMatchGetByIdResponse(
	match *entity.Match,
) *dto.MatchGetByIdResponse {
	gamesResponse := []*dto.GameResponse{}

	for _, game := range match.Games {
		gamesResponse = append(
			gamesResponse,
			&dto.GameResponse{
				ID:                  game.ID,
				CreatedAt:           game.CreatedAt,
				MatchId:             match.ID,
				UserId:              match.UserId,
				GoFirst:             game.GoFirst,
				WinningFlg:          game.WinningFlg,
				YourPrizeCards:      game.YourPrizeCards,
				OpponentsPrizeCards: game.OpponentsPrizeCards,
				Memo:                game.Memo,
			},
		)
	}

	return &dto.MatchGetByIdResponse{
		MatchResponse: dto.MatchResponse{
			ID:                 match.ID,
			CreatedAt:          match.CreatedAt,
			RecordId:           match.RecordId,
			DeckId:             match.DeckId,
			DeckCodeId:         match.DeckCodeId,
			UserId:             match.UserId,
			OpponentsUserId:    match.OpponentsUserId,
			BO3Flg:             match.BO3Flg,
			QualifyingRoundFlg: match.QualifyingRoundFlg,
			FinalTournamentFlg: match.FinalTournamentFlg,
			DefaultVictoryFlg:  match.DefaultVictoryFlg,
			DefaultDefeatFlg:   match.DefaultDefeatFlg,
			VictoryFlg:         match.VictoryFlg,
			OpponentsDeckInfo:  match.OpponentsDeckInfo,
			Memo:               match.Memo,
			Games:              gamesResponse,
		},
	}
}

func NewMatchGetByRecordIdResponse(
	matches []*entity.Match,
) []*dto.MatchResponse {
	matchesResponse := []*dto.MatchResponse{}

	for _, match := range matches {
		gamesResponse := []*dto.GameResponse{}
		for _, game := range match.Games {
			gamesResponse = append(
				gamesResponse,
				&dto.GameResponse{
					ID:                  game.ID,
					CreatedAt:           game.CreatedAt,
					MatchId:             match.ID,
					UserId:              match.UserId,
					GoFirst:             game.GoFirst,
					WinningFlg:          game.WinningFlg,
					YourPrizeCards:      game.YourPrizeCards,
					OpponentsPrizeCards: game.OpponentsPrizeCards,
					Memo:                game.Memo,
				},
			)
		}

		matchesResponse = append(
			matchesResponse,
			&dto.MatchResponse{
				ID:                 match.ID,
				CreatedAt:          match.CreatedAt,
				RecordId:           match.RecordId,
				DeckId:             match.DeckId,
				DeckCodeId:         match.DeckCodeId,
				UserId:             match.UserId,
				OpponentsUserId:    match.OpponentsUserId,
				BO3Flg:             match.BO3Flg,
				QualifyingRoundFlg: match.QualifyingRoundFlg,
				FinalTournamentFlg: match.FinalTournamentFlg,
				DefaultVictoryFlg:  match.DefaultVictoryFlg,
				DefaultDefeatFlg:   match.DefaultDefeatFlg,
				VictoryFlg:         match.VictoryFlg,
				OpponentsDeckInfo:  match.OpponentsDeckInfo,
				Memo:               match.Memo,
				Games:              gamesResponse,
			},
		)
	}

	return matchesResponse
}

func NewMatchCreateResponse(
	match *entity.Match,
) *dto.MatchCreateResponse {
	gamesResponse := []*dto.GameResponse{}

	for _, game := range match.Games {
		gamesResponse = append(
			gamesResponse,
			&dto.GameResponse{
				ID:                  game.ID,
				CreatedAt:           game.CreatedAt,
				MatchId:             match.ID,
				UserId:              match.UserId,
				GoFirst:             game.GoFirst,
				WinningFlg:          game.WinningFlg,
				YourPrizeCards:      game.YourPrizeCards,
				OpponentsPrizeCards: game.OpponentsPrizeCards,
				Memo:                game.Memo,
			},
		)
	}

	return &dto.MatchCreateResponse{
		MatchResponse: dto.MatchResponse{
			ID:                 match.ID,
			CreatedAt:          match.CreatedAt,
			RecordId:           match.RecordId,
			DeckId:             match.DeckId,
			DeckCodeId:         match.DeckCodeId,
			UserId:             match.UserId,
			OpponentsUserId:    match.OpponentsUserId,
			BO3Flg:             match.BO3Flg,
			QualifyingRoundFlg: match.QualifyingRoundFlg,
			FinalTournamentFlg: match.FinalTournamentFlg,
			DefaultVictoryFlg:  match.DefaultVictoryFlg,
			DefaultDefeatFlg:   match.DefaultDefeatFlg,
			VictoryFlg:         match.VictoryFlg,
			OpponentsDeckInfo:  match.OpponentsDeckInfo,
			Memo:               match.Memo,
			Games:              gamesResponse,
		},
	}
}

func NewMatchUpdateResponse(
	match *entity.Match,
) *dto.MatchUpdateResponse {
	gamesResponse := []*dto.GameResponse{}

	for _, game := range match.Games {
		gamesResponse = append(
			gamesResponse,
			&dto.GameResponse{
				ID:                  game.ID,
				CreatedAt:           game.CreatedAt,
				MatchId:             match.ID,
				UserId:              match.UserId,
				GoFirst:             game.GoFirst,
				WinningFlg:          game.WinningFlg,
				YourPrizeCards:      game.YourPrizeCards,
				OpponentsPrizeCards: game.OpponentsPrizeCards,
				Memo:                game.Memo,
			},
		)
	}

	return &dto.MatchUpdateResponse{
		MatchResponse: dto.MatchResponse{
			ID:                 match.ID,
			CreatedAt:          match.CreatedAt,
			RecordId:           match.RecordId,
			DeckId:             match.DeckId,
			DeckCodeId:         match.DeckCodeId,
			UserId:             match.UserId,
			OpponentsUserId:    match.OpponentsUserId,
			BO3Flg:             match.BO3Flg,
			QualifyingRoundFlg: match.QualifyingRoundFlg,
			FinalTournamentFlg: match.FinalTournamentFlg,
			DefaultVictoryFlg:  match.DefaultVictoryFlg,
			DefaultDefeatFlg:   match.DefaultDefeatFlg,
			VictoryFlg:         match.VictoryFlg,
			OpponentsDeckInfo:  match.OpponentsDeckInfo,
			Memo:               match.Memo,
			Games:              gamesResponse,
		},
	}
}
