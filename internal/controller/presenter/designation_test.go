package presenter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func newTestDesignationEntity(criteriaType string) *entity.Designation {
	now := time.Now().Local()
	return entity.NewDesignation(
		"designation-regular", 4, "regular", "🎖", "レギュラー", "シティリーグ出場",
		criteriaType, 1, now, now,
	)
}

func TestNewDesignationsResponse(t *testing.T) {
	t.Run("正常系_シティリーグ条件の称号には単独達成の閾値が付与される", func(t *testing.T) {
		res := NewDesignationsResponse([]*entity.Designation{
			newTestDesignationEntity(usecase.DesignationCriteriaTypeOfficialCityLeagueRecord),
		})

		require.Len(t, res.Designations, 1)
		require.Equal(t, usecase.DesignationCityLeagueStandaloneThreshold, res.Designations[0].StandaloneThreshold)
	})

	t.Run("正常系_その他の条件の称号は閾値0のまま返す", func(t *testing.T) {
		res := NewDesignationsResponse([]*entity.Designation{
			newTestDesignationEntity("record_count"),
		})

		require.Len(t, res.Designations, 1)
		require.Zero(t, res.Designations[0].StandaloneThreshold)
	})
}

func TestNewUserDesignationResponse(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_現在の称号とロードマップを変換して返す", func(t *testing.T) {
		view := &usecase.UserDesignationView{
			Current: newTestDesignationEntity("record_count"),
			Ladder: []*usecase.DesignationLadderItem{
				{
					Designation:  newTestDesignationEntity("record_count"),
					Achieved:     true,
					CurrentValue: 5,
				},
			},
		}

		res := NewUserDesignationResponse(uid, "2026", view)

		require.Equal(t, uid, res.UserId)
		require.Equal(t, "2026", res.Season)
		require.NotNil(t, res.Current)
		require.Equal(t, "designation-regular", res.Current.ID)
		require.Len(t, res.Ladder, 1)
		require.True(t, res.Ladder[0].Achieved)
		require.Equal(t, 5, res.Ladder[0].CurrentValue)
	})

	t.Run("正常系_称号未達成ならcurrentはnullになる", func(t *testing.T) {
		res := NewUserDesignationResponse(uid, "2026", &usecase.UserDesignationView{})

		require.Nil(t, res.Current)
		require.Empty(t, res.Ladder)
	})
}

func TestNewDesignationRankStatsResponse(t *testing.T) {
	t.Run("正常系_ティア別の到達ユーザー数を変換して返す", func(t *testing.T) {
		view := &usecase.DesignationRankStatsView{
			TotalUsers: 10,
			Tiers: []*usecase.DesignationTierStat{
				{Tier: 1, UserCount: 7},
				{Tier: 5, UserCount: 3},
			},
		}

		res := NewDesignationRankStatsResponse("2026", view)

		require.Equal(t, "2026", res.Season)
		require.Equal(t, 10, res.TotalUsers)
		require.Len(t, res.Tiers, 2)
		require.Equal(t, 1, res.Tiers[0].Tier)
		require.Equal(t, 7, res.Tiers[0].UserCount)
	})
}
