package presenter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func TestNewWeeklyDeckUsageStatResponse(t *testing.T) {
	weekStart := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)

	t.Run("正常系_週の開始日と終了日をYYYY-MM-DD形式で返す", func(t *testing.T) {
		stat := entity.NewWeeklyDeckUsageStat(weekStart, 0, 0, []*entity.DeckUsageVariant{})

		res := NewWeeklyDeckUsageStatResponse(stat, "2026-07-16")

		require.Equal(t, "2026-07-16", res.Week)
		require.Equal(t, "2026-07-13", res.WeekStart)
		// 終了日は開始日+6日(週の最終日)
		require.Equal(t, "2026-07-19", res.WeekEnd)
	})

	t.Run("正常系_変種ごとの集計値とスプライトを変換する", func(t *testing.T) {
		variant := entity.NewDeckUsageVariant(
			"fingerprint-1", 5, 0.5, 3, 2, 0.6,
			[]*entity.PokemonSprite{entity.NewPokemonSprite("pikachu")},
		)
		stat := entity.NewWeeklyDeckUsageStat(weekStart, 10, 3, []*entity.DeckUsageVariant{variant})

		res := NewWeeklyDeckUsageStatResponse(stat, "")

		require.Equal(t, 10, res.TotalVotes)
		require.Equal(t, 3, res.ContributorCount)
		require.Len(t, res.Decks, 1)
		require.Equal(t, "fingerprint-1", res.Decks[0].Fingerprint)
		require.Equal(t, 5, res.Decks[0].Count)
		require.InDelta(t, 0.6, res.Decks[0].WinRate, 1e-9)
		require.Len(t, res.Decks[0].PokemonSprites, 1)
		require.Equal(t, "pikachu", res.Decks[0].PokemonSprites[0].ID)
	})
}
