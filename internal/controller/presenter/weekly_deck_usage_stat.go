package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

const weekDateLayout = "2006-01-02"

func NewWeeklyDeckUsageStatResponse(
	stat *entity.WeeklyDeckUsageStat,
	week string,
) *dto.WeeklyDeckUsageStatResponse {
	decks := []*dto.WeeklyDeckUsageItemResponse{}
	for _, deck := range stat.Decks {
		decks = append(decks, newWeeklyDeckUsageItemResponse(deck))
	}

	// WeekStart は月曜0時。WeekEnd は表示用に週末（日曜）を返す。
	weekStart := stat.WeekStart.Format(weekDateLayout)
	weekEnd := stat.WeekStart.AddDate(0, 0, 6).Format(weekDateLayout)

	return &dto.WeeklyDeckUsageStatResponse{
		Week:             week,
		WeekStart:        weekStart,
		WeekEnd:          weekEnd,
		TotalVotes:       stat.TotalVotes,
		ContributorCount: stat.ContributorCount,
		Decks:            decks,
	}
}

// newWeeklyDeckUsageItemResponse は変種 1 件を DTO へ変換する。
// 「その他」枠の内訳（Members）も同じ構造で再帰的に変換する。
func newWeeklyDeckUsageItemResponse(
	deck *entity.DeckUsageVariant,
) *dto.WeeklyDeckUsageItemResponse {
	pokemonSprites := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deck.PokemonSprites {
		pokemonSprites = append(pokemonSprites, &dto.PokemonSpriteResponse{
			ID:       pokemonSprite.ID,
			Position: pokemonSprite.Position,
		})
	}

	var members []*dto.WeeklyDeckUsageItemResponse
	for _, member := range deck.Members {
		members = append(members, newWeeklyDeckUsageItemResponse(member))
	}

	return &dto.WeeklyDeckUsageItemResponse{
		Fingerprint:    deck.Fingerprint,
		Count:          deck.Count,
		UsageRate:      deck.UsageRate,
		Wins:           deck.Wins,
		Losses:         deck.Losses,
		WinRate:        deck.WinRate,
		PokemonSprites: pokemonSprites,
		Members:        members,
	}
}
