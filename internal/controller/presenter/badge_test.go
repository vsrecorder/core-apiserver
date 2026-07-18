package presenter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func newTestBadgeDefinitionEntity() *entity.BadgeDefinition {
	now := time.Now().Local()
	return entity.NewBadgeDefinition(
		"badge-first-record", "first_record", "onboarding", "はじめての記録", "初めて記録を作成した",
		"icon_first_record", "record_count", 1, now, time.Time{}, now, now,
	)
}

func TestNewBadgeDefinitionsResponse(t *testing.T) {
	t.Run("正常系_バッジ定義を変換して返す", func(t *testing.T) {
		res := NewBadgeDefinitionsResponse([]*entity.BadgeDefinition{newTestBadgeDefinitionEntity()})

		require.Len(t, res.Badges, 1)
		require.Equal(t, "badge-first-record", res.Badges[0].ID)
		require.Equal(t, "first_record", res.Badges[0].Code)
		require.Equal(t, "record_count", res.Badges[0].CriteriaType)
		require.Equal(t, 1, res.Badges[0].CriteriaValue)
	})
}

func TestNewUserBadgesResponse(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_獲得済みバッジは獲得日時が設定される", func(t *testing.T) {
		achievedAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
		views := []*usecase.UserBadgeView{
			{Definition: newTestBadgeDefinitionEntity(), Achieved: true, AchievedAt: achievedAt, CurrentValue: 3},
		}

		res := NewUserBadgesResponse(uid, "2026", views)

		require.Equal(t, uid, res.UserId)
		require.Equal(t, "2026", res.Season)
		require.Len(t, res.Badges, 1)
		require.True(t, res.Badges[0].Achieved)
		require.NotNil(t, res.Badges[0].AchievedAt)
		require.Equal(t, achievedAt, *res.Badges[0].AchievedAt)
		require.Equal(t, 3, res.Badges[0].CurrentValue)
	})

	t.Run("正常系_未獲得バッジは獲得日時がnullになる", func(t *testing.T) {
		views := []*usecase.UserBadgeView{
			{Definition: newTestBadgeDefinitionEntity(), Achieved: false},
		}

		res := NewUserBadgesResponse(uid, "2026", views)

		require.Len(t, res.Badges, 1)
		require.False(t, res.Badges[0].Achieved)
		require.Nil(t, res.Badges[0].AchievedAt)
	})
}
