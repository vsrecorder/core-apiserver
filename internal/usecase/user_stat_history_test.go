package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestUserStatHistoryUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserStatHistoryInterface(mockCtrl)
	usecase := NewUserStatHistory(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockUserStatHistoryInterface,
		usecase UserStatHistoryInterface,
	){
		"GetUserStatHistory": test_UserStatHistoryUsecase_GetUserStatHistory,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_UserStatHistoryUsecase_GetUserStatHistory(t *testing.T, mockRepository *mock_repository.MockUserStatHistoryInterface, usecase UserStatHistoryInterface) {
	t.Run("正常系_#01_deck_id指定時はそのままrepositoryに渡される", func(t *testing.T) {
		userId := "user-01"
		period := "3months"
		season := ""
		deckId := "deck-01"

		history := []*entity.UserStatMonthly{
			entity.NewUserStatMonthly("2026-06", 10, 6, 4, 0.6),
		}

		mockRepository.EXPECT().
			FindUserStatHistory(context.Background(), userId, gomock.Any(), gomock.Any(), deckId).
			Return(history, nil)

		ret, err := usecase.GetUserStatHistory(context.Background(), userId, period, season, deckId)

		require.NoError(t, err)
		require.Equal(t, history, ret)
	})

	t.Run("正常系_#02_deck_id未指定でも空文字のまま渡される", func(t *testing.T) {
		userId := "user-02"
		period := "6months"
		season := ""
		deckId := ""

		mockRepository.EXPECT().
			FindUserStatHistory(context.Background(), userId, gomock.Any(), gomock.Any(), deckId).
			Return([]*entity.UserStatMonthly{}, nil)

		ret, err := usecase.GetUserStatHistory(context.Background(), userId, period, season, deckId)

		require.NoError(t, err)
		require.Empty(t, ret)
	})
}
