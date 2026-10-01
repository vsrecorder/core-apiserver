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

func NewOpponentDeckReplaceResponse(
	updatedCount int,
) *dto.OpponentDeckReplaceResponse {
	return &dto.OpponentDeckReplaceResponse{
		UpdatedCount: updatedCount,
	}
}
