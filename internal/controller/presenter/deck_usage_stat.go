package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewDeckUsageStatResponse(
	stat *entity.DeckUsageStat,
	yearMonth string,
	environmentId string,
	season string,
	regulationId string,
) *dto.DeckUsageStatResponse {
	decks := []*dto.DeckUsageItemResponse{}
	for _, deck := range stat.Decks {
		pokemonSprites := []*dto.PokemonSpriteResponse{}
		for _, pokemonSprite := range deck.PokemonSprites {
			pokemonSprites = append(pokemonSprites, &dto.PokemonSpriteResponse{
				ID: pokemonSprite.ID,
			})
		}

		decks = append(decks, &dto.DeckUsageItemResponse{
			DeckId:         deck.DeckId,
			Name:           deck.Name,
			Count:          deck.Count,
			UsageRate:      deck.UsageRate,
			Wins:           deck.Wins,
			Losses:         deck.Losses,
			WinRate:        deck.WinRate,
			PokemonSprites: pokemonSprites,
		})
	}

	return &dto.DeckUsageStatResponse{
		UserId:        stat.UserId,
		YearMonth:     yearMonth,
		EnvironmentId: environmentId,
		Season:        season,
		RegulationId:  regulationId,
		TotalRecords:  stat.TotalRecords,
		Decks:         decks,
	}
}
