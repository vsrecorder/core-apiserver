package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func newBadgeTestUsecase(mockCtrl *gomock.Controller) (
	*Badge,
	*mock_repository.MockBadgeDefinitionInterface,
	*mock_repository.MockUserBadgeInterface,
	*mock_repository.MockBadgeStatsInterface,
) {
	badgeDefinitionRepo := mock_repository.NewMockBadgeDefinitionInterface(mockCtrl)
	userBadgeRepo := mock_repository.NewMockUserBadgeInterface(mockCtrl)
	badgeStatsRepo := mock_repository.NewMockBadgeStatsInterface(mockCtrl)

	u := &Badge{
		badgeDefinitionRepo: badgeDefinitionRepo,
		userBadgeRepo:       userBadgeRepo,
		badgeStatsRepo:      badgeStatsRepo,
	}

	return u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo
}

func findView(views []*UserBadgeView, id string) *UserBadgeView {
	for _, v := range views {
		if v.Definition.ID == id {
			return v
		}
	}
	return nil
}

func TestBadge_GetByUserId(t *testing.T) {
	t.Run("オンボーディング系は永続化された獲得記録をそのまま参照する(シーズン集計値は使わない)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", BadgeCategoryOnboarding, "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-first-record", "record-1", now),
			}, nil,
		)
		// オンボーディングは全期間、マイルストーン/ストリークは今シーズンの2種類の集計値を取得する
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-first-record")
		require.NotNil(t, view)
		require.True(t, view.Achieved)
		require.Equal(t, now.Unix(), view.AchievedAt.Unix())
	})

	t.Run("マイルストーン系は今シーズンの集計値のみでライブ判定する(過去の獲得記録は見ない)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", BadgeCategoryMilestone, "駆け出しユーザー", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		// user_badges には何も無い(永続化していない)が、今シーズンの記録数が10件あるので達成扱いになる
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		// 全期間カウント(オンボーディング用、呼ばれるが今回は未使用)
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(999, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)

		// 今シーズンのカウント(マイルストーン用)
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Not(time.Time{})).Return(10, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Not(time.Time{})).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Not(time.Time{})).Return(0, nil)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-record-10")
		require.NotNil(t, view)
		require.True(t, view.Achieved)
		require.Equal(t, 10, view.CurrentValue)
		require.True(t, view.AchievedAt.IsZero(), "ライブ判定のバッジはachieved_atを持たない")
	})

	t.Run("週次ストリーク系は今シーズンの記録日から連続週数を計算して判定する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", BadgeCategoryStreak, "週次記録3週連続", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		// 3週連続分の記録日(月曜始まり基準で3週分)
		week1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		week2 := time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local)
		week3 := time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).
			Return([]time.Time{week1, week2, week3}, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-streak-3")
		require.NotNil(t, view)
		require.True(t, view.Achieved)
		require.Equal(t, 3, view.CurrentValue)
	})

	t.Run("season指定時はそのシーズンの期間で集計する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", BadgeCategoryMilestone, "駆け出しユーザー", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}
		wantFrom := time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local)
		wantTo := time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)

		// 2024シーズン(2023-09-01〜2024-08-31)がそのまま渡されることを検証する
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(3, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(0, nil)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "2024")

		require.NoError(t, err)
		view := findView(views, "def-record-10")
		require.NotNil(t, view)
		require.Equal(t, 3, view.CurrentValue)
	})
}

func TestSeasonStreakWeeks(t *testing.T) {
	t.Run("記録が無ければ0", func(t *testing.T) {
		require.Equal(t, 0, seasonStreakWeeks(nil))
	})

	t.Run("連続した週なら連続数がそのまま返る", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		require.Equal(t, 3, seasonStreakWeeks(dates))
	})

	t.Run("1週空いてもフリーズ枠で連続扱いになる", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local), // 2週間後(1週間の空白)
		}
		require.Equal(t, 2, seasonStreakWeeks(dates))
	})

	t.Run("フリーズ枠を超えて空くと直近の連続数だけが残る", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local), // 大きく空白
			time.Date(2026, 7, 27, 0, 0, 0, 0, time.Local),
		}
		require.Equal(t, 2, seasonStreakWeeks(dates))
	})
}
