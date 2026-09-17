package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewOpponentDeckCandidatesGetResponse(
	limit int,
	candidates []*entity.OpponentDeckCandidate,
) *dto.OpponentDeckCandidatesGetResponse {
	data := make([]*dto.OpponentDeckCandidateResponse, 0, len(candidates))
	for _, candidate := range candidates {
		sprites := make([]*dto.PokemonSpriteResponse, 0, len(candidate.PokemonSprites))
		for _, sprite := range candidate.PokemonSprites {
			sprites = append(sprites, &dto.PokemonSpriteResponse{
				ID:       sprite.ID,
				Position: sprite.Position,
			})
		}

		data = append(data, &dto.OpponentDeckCandidateResponse{
			OpponentsDeckInfo: candidate.OpponentsDeckInfo,
			PokemonSprites:    sprites,
			Count:             candidate.Count,
		})
	}

	return &dto.OpponentDeckCandidatesGetResponse{
		Limit: limit,
		Data:  data,
	}
}
