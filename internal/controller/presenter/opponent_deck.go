package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewOpponentDecksGetResponse(
	decks []*entity.OpponentDeck,
) *dto.OpponentDecksGetResponse {
	data := make([]*dto.OpponentDeckResponse, 0, len(decks))
	for _, deck := range decks {
		sprites := make([]*dto.PokemonSpriteResponse, 0, len(deck.PokemonSprites))
		for _, sprite := range deck.PokemonSprites {
			sprites = append(sprites, &dto.PokemonSpriteResponse{
				ID:       sprite.ID,
				Position: sprite.Position,
			})
		}

		data = append(data, &dto.OpponentDeckResponse{
			OpponentsDeckInfo: deck.OpponentsDeckInfo,
			PokemonSprites:    sprites,
			Count:             deck.Count,
			LastEventDate:     deck.LastEventDate,
		})
	}

	return &dto.OpponentDecksGetResponse{
		Data: data,
	}
}

func NewOpponentDeckMatchesGetResponse(
	matches []*entity.OpponentDeckMatch,
) *dto.OpponentDeckMatchesGetResponse {
	data := make([]*dto.OpponentDeckMatchResponse, 0, len(matches))
	for _, match := range matches {
		games := make([]*dto.OpponentDeckMatchGameResponse, 0, len(match.Games))
		for _, game := range match.Games {
			games = append(games, &dto.OpponentDeckMatchGameResponse{
				GoFirst:    game.GoFirst,
				WinningFlg: game.WinningFlg,
			})
		}

		data = append(data, &dto.OpponentDeckMatchResponse{
			ID:                   match.MatchId,
			RecordId:             match.RecordId,
			EventDate:            match.EventDate,
			EventType:            match.EventType,
			EventTitle:           match.EventTitle,
			DeckName:             match.DeckName,
			BO3Flg:               match.BO3Flg,
			GroupMatchFlg:        match.GroupMatchFlg,
			GroupMatchVictoryFlg: match.GroupMatchVictoryFlg,
			DefaultVictoryFlg:    match.DefaultVictoryFlg,
			DefaultDefeatFlg:     match.DefaultDefeatFlg,
			VictoryFlg:           match.VictoryFlg,
			DrawFlg:              match.DrawFlg,
			Games:                games,
		})
	}

	return &dto.OpponentDeckMatchesGetResponse{
		Data: data,
	}
}

func NewOpponentDeckReplaceResponse(
	updatedCount int,
) *dto.OpponentDeckReplaceResponse {
	return &dto.OpponentDeckReplaceResponse{
		UpdatedCount: updatedCount,
	}
}
