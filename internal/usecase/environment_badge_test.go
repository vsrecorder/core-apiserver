package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func setup4EnvironmentBadgeUsecase(t *testing.T) (
	*mock_repository.MockEnvironmentInterface,
	*mock_repository.MockUserEnvironmentBadgeInterface,
	EnvironmentBadgeInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockEnvironmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	mockUserEnvironmentBadgeRepo := mock_repository.NewMockUserEnvironmentBadgeInterface(mockCtrl)

	usecase := NewEnvironmentBadge(mockEnvironmentRepo, mockUserEnvironmentBadgeRepo)

	return mockEnvironmentRepo, mockUserEnvironmentBadgeRepo, usecase
}

func TestEnvironmentBadgeUsecase_GetByUserId(t *testing.T) {
	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_獲得済み環境にはAchievedと獲得日時が設定される", func(t *testing.T) {
		mockEnvironmentRepo, mockUserEnvironmentBadgeRepo, usecase := setup4EnvironmentBadgeUsecase(t)

		environments := []*entity.Environment{
			{ID: "sv10"},
			{ID: "sv11"},
		}

		achievedAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local)
		userBadges := []*entity.UserEnvironmentBadge{
			{EnvironmentId: "sv11", AchievedAt: achievedAt},
		}

		mockEnvironmentRepo.EXPECT().Find(context.Background()).Return(environments, nil)
		mockUserEnvironmentBadgeRepo.EXPECT().FindByUserId(context.Background(), userId).Return(userBadges, nil)

		views, err := usecase.GetByUserId(context.Background(), userId)

		require.NoError(t, err)
		require.Len(t, views, 2)

		require.Equal(t, "sv10", views[0].Environment.ID)
		require.False(t, views[0].Achieved)
		require.True(t, views[0].AchievedAt.IsZero())

		require.Equal(t, "sv11", views[1].Environment.ID)
		require.True(t, views[1].Achieved)
		require.Equal(t, achievedAt, views[1].AchievedAt)
	})

	t.Run("正常系_獲得なしでも全環境が未獲得として返る", func(t *testing.T) {
		mockEnvironmentRepo, mockUserEnvironmentBadgeRepo, usecase := setup4EnvironmentBadgeUsecase(t)

		environments := []*entity.Environment{{ID: "sv10"}}

		mockEnvironmentRepo.EXPECT().Find(context.Background()).Return(environments, nil)
		mockUserEnvironmentBadgeRepo.EXPECT().FindByUserId(context.Background(), userId).Return(nil, nil)

		views, err := usecase.GetByUserId(context.Background(), userId)

		require.NoError(t, err)
		require.Len(t, views, 1)
		require.False(t, views[0].Achieved)
	})

	t.Run("異常系_環境一覧の取得エラーをそのまま返す", func(t *testing.T) {
		mockEnvironmentRepo, _, usecase := setup4EnvironmentBadgeUsecase(t)

		mockEnvironmentRepo.EXPECT().Find(context.Background()).Return(nil, errors.New(""))

		views, err := usecase.GetByUserId(context.Background(), userId)

		require.Error(t, err)
		require.Nil(t, views)
	})

	t.Run("異常系_獲得状況の取得エラーをそのまま返す", func(t *testing.T) {
		mockEnvironmentRepo, mockUserEnvironmentBadgeRepo, usecase := setup4EnvironmentBadgeUsecase(t)

		mockEnvironmentRepo.EXPECT().Find(context.Background()).Return([]*entity.Environment{}, nil)
		mockUserEnvironmentBadgeRepo.EXPECT().FindByUserId(context.Background(), userId).Return(nil, errors.New(""))

		views, err := usecase.GetByUserId(context.Background(), userId)

		require.Error(t, err)
		require.Nil(t, views)
	})
}
