package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewDeckUsageStatResponse(
	stat *entity.DeckUsageStat,
	week string,
	yearMonth string,
	environmentId string,
	season string,
	standardRegulationId string,
	regulationId uint,
) *dto.DeckUsageStatResponse {
	decks := []*dto.DeckUsageItemResponse{}
	for _, deck := range stat.Decks {
		pokemonSprites := []*dto.PokemonSpriteResponse{}
		for _, pokemonSprite := range deck.PokemonSprites {
			pokemonSprites = append(pokemonSprites, &dto.PokemonSpriteResponse{
				ID:       pokemonSprite.ID,
				Position: pokemonSprite.Position,
			})
		}

		decks = append(decks, &dto.DeckUsageItemResponse{
			DeckId:          deck.DeckId,
			Name:            deck.Name,
			Count:           deck.Count,
			UsageRate:       deck.UsageRate,
			Wins:            deck.Wins,
			Losses:          deck.Losses,
			WinRate:         deck.WinRate,
			GameCount:       deck.GameCount,
			GoFirstCount:    deck.GoFirstCount,
			GoSecondCount:   deck.GoSecondCount,
			GoFirstRate:     deck.GoFirstRate,
			GoFirstWins:     deck.GoFirstWins,
			GoFirstWinRate:  deck.GoFirstWinRate,
			GoSecondWins:    deck.GoSecondWins,
			GoSecondWinRate: deck.GoSecondWinRate,
			IgnoredCount:    deck.IgnoredCount,
			PokemonSprites:  pokemonSprites,
		})
	}

	return &dto.DeckUsageStatResponse{
		UserId:               stat.UserId,
		Week:                 week,
		YearMonth:            yearMonth,
		EnvironmentId:        environmentId,
		Season:               season,
		StandardRegulationId: standardRegulationId,
		RegulationId:         regulationId,
		TotalRecords:         stat.TotalRecords,
		Decks:                decks,
	}
}
