package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewOpponentDeckUsageStatResponse(
	stat *entity.OpponentDeckUsageStat,
	yearMonth string,
	environmentId string,
	season string,
	regulationId string,
	deckId string,
) *dto.OpponentDeckUsageStatResponse {
	decks := []*dto.OpponentDeckUsageItemResponse{}
	for _, deck := range stat.Decks {
		pokemonSprites := []*dto.PokemonSpriteResponse{}
		for _, pokemonSprite := range deck.PokemonSprites {
			pokemonSprites = append(pokemonSprites, &dto.PokemonSpriteResponse{
				ID:       pokemonSprite.ID,
				Position: pokemonSprite.Position,
			})
		}

		decks = append(decks, &dto.OpponentDeckUsageItemResponse{
			DeckInfo:       deck.DeckInfo,
			Count:          deck.Count,
			UsageRate:      deck.UsageRate,
			Wins:           deck.Wins,
			Losses:         deck.Losses,
			WinRate:        deck.WinRate,
			PokemonSprites: pokemonSprites,
		})
	}

	return &dto.OpponentDeckUsageStatResponse{
		UserId:        stat.UserId,
		YearMonth:     yearMonth,
		EnvironmentId: environmentId,
		Season:        season,
		RegulationId:  regulationId,
		DeckId:        deckId,
		TotalMatches:  stat.TotalMatches,
		Decks:         decks,
	}
}
