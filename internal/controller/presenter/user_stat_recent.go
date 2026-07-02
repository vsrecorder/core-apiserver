package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewRecentMatchStatResponse(
	stat *entity.RecentMatchStat,
	deckId string,
) *dto.RecentMatchStatResponse {
	matches := make([]dto.RecentMatchItem, 0, len(stat.Matches))
	for _, m := range stat.Matches {
		pokemonSprites := make([]*dto.PokemonSpriteResponse, 0, len(m.PokemonSprites))
		for _, sprite := range m.PokemonSprites {
			pokemonSprites = append(pokemonSprites, &dto.PokemonSpriteResponse{
				ID: sprite.ID,
			})
		}

		matches = append(matches, dto.RecentMatchItem{
			Sequence:          m.Sequence,
			EventDate:         m.EventDate.Format("2006-01-02"),
			DeckId:            m.DeckId,
			OpponentsDeckInfo: m.OpponentsDeckInfo,
			Victory:           m.VictoryFlg,
			RollingWinRate:    m.RollingWinRate,
			EnvironmentId:     m.EnvironmentId,
			EnvironmentTitle:  m.EnvironmentTitle,
			PokemonSprites:    pokemonSprites,
		})
	}

	return &dto.RecentMatchStatResponse{
		UserId:       stat.UserId,
		Count:        stat.Count,
		DeckId:       deckId,
		TotalMatches: stat.TotalMatches,
		Wins:         stat.Wins,
		WinRate:      stat.WinRate,
		Matches:      matches,
	}
}
