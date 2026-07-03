package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func newDesignationTestUsecase(mockCtrl *gomock.Controller) (
	*Designation,
	*mock_repository.MockDesignationInterface,
	*mock_repository.MockDesignationStatsInterface,
) {
	designationRepo := mock_repository.NewMockDesignationInterface(mockCtrl)
	designationStatsRepo := mock_repository.NewMockDesignationStatsInterface(mockCtrl)

	u := &Designation{designationRepo, designationStatsRepo}

	return u, designationRepo, designationStatsRepo
}

// threeTierDefinitions は 駆け出し(tier1, ジムバトル1件)・見習い(tier2, ジムバトル5件)・
// 一人前(tier3, 見習いの条件+リーグ記録)という累積構造を再現した3ティア。
func threeTierDefinitions(now time.Time) []*entity.Designation {
	return []*entity.Designation{
		entity.NewDesignation("designation-01", 1, "beginner", "🌱", "駆け出し", "", DesignationCriteriaTypeOfficialGymBattleRecord, 1, now, now),
		entity.NewDesignation("designation-02", 2, "novice", "🔰", "見習い", "", DesignationCriteriaTypeOfficialGymBattleRecord, 5, now, now),
		entity.NewDesignation("designation-03", 3, "independent", "👍", "一人前", "", DesignationCriteriaTypeOfficialLeagueRecord, 1, now, now),
	}
}

// fourTierDefinitions は threeTierDefinitions に、常連(tier4, シティリーグ記録4件)を
// 加えた4ティア。
func fourTierDefinitions(now time.Time) []*entity.Designation {
	return append(
		threeTierDefinitions(now),
		entity.NewDesignation("designation-04", 4, "regular", "🎫", "常連", "", DesignationCriteriaTypeOfficialCityLeagueRecord, 4, now, now),
	)
}

func TestDesignation_GetByUserId(t *testing.T) {
	t.Run("今シーズンの集計値が条件を満たすと現在の称号として返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo := newDesignationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountGymBattleRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-02", view.Current.ID)

		// ロードマップは tier1・tier2 が達成扱い、tier3 は未達成のまま
		for _, item := range view.Ladder {
			switch item.Designation.ID {
			case "designation-01", "designation-02":
				require.True(t, item.Achieved, item.Designation.ID)
				require.Equal(t, 5, item.CurrentValue, item.Designation.ID)
			case "designation-03":
				require.False(t, item.Achieved, item.Designation.ID)
				require.Equal(t, 0, item.CurrentValue, item.Designation.ID)
			}
		}
	})

	t.Run("シーズンが変わり集計値が0に戻れば称号なしになる(永続化された過去の実績は見ない)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo := newDesignationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// 新シーズンでまだ何も記録していない
		designationStatsRepo.EXPECT().CountGymBattleRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.Nil(t, view.Current)
		for _, item := range view.Ladder {
			require.False(t, item.Achieved, item.Designation.ID)
		}
	})

	t.Run("未実装(準備中)のティアは絶対に達成されない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo := newDesignationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.Designation{
			entity.NewDesignation("designation-04", 4, "regular", "🎫", "常連", "", "unimplemented", 0, now, now),
		}
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		designationStatsRepo.EXPECT().CountGymBattleRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(999, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(999, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(999, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.Nil(t, view.Current)
	})

	t.Run("シティリーグ記録数が条件を満たすと常連(tier4)まで到達する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo := newDesignationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountGymBattleRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(4, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.True(t, item04.Achieved)
		require.Equal(t, 4, item04.CurrentValue)
	})

	t.Run("シティリーグ記録数が不足していると常連(tier4)には到達しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo := newDesignationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountGymBattleRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		// トレーナーズリーグの記録はあるが、シティリーグ単独では3件しかない
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(3, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-03", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.False(t, item04.Achieved)
		require.Equal(t, 3, item04.CurrentValue)
	})
}

func findDesignationLadderItem(ladder []*DesignationLadderItem, id string) *DesignationLadderItem {
	for _, item := range ladder {
		if item.Designation.ID == id {
			return item
		}
	}
	return nil
}
