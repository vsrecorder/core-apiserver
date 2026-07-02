package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestOpponentDeckUsageStatUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockOpponentDeckUsageStatInterface(mockCtrl)
	mockEnvironmentRepository := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	usecase := NewOpponentDeckUsageStat(mockRepository, mockEnvironmentRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockOpponentDeckUsageStatInterface,
		usecase OpponentDeckUsageStatInterface,
	){
		"GetOpponentDeckUsageStat": test_OpponentDeckUsageStatUsecase_GetOpponentDeckUsageStat,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_OpponentDeckUsageStatUsecase_GetOpponentDeckUsageStat(t *testing.T, mockRepository *mock_repository.MockOpponentDeckUsageStatInterface, usecase OpponentDeckUsageStatInterface) {
	t.Run("正常系_#01_deck_id指定時はそのままrepositoryに渡される", func(t *testing.T) {
		userId := "user-01"
		yearMonth := "2026-06"
		environmentId := ""
		season := ""
		deckId := "deck-01"

		stat := entity.NewOpponentDeckUsageStat(userId, 5, []*entity.OpponentDeckUsage{})

		mockRepository.EXPECT().
			FindOpponentDeckUsageStat(context.Background(), userId, gomock.Any(), gomock.Any(), deckId).
			Return(stat, nil)

		ret, err := usecase.GetOpponentDeckUsageStat(context.Background(), userId, yearMonth, environmentId, season, deckId)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_#02_deck_id未指定でも空文字のまま渡される", func(t *testing.T) {
		userId := "user-02"
		yearMonth := "2026-06"
		environmentId := ""
		season := ""
		deckId := ""

		stat := entity.NewOpponentDeckUsageStat(userId, 0, []*entity.OpponentDeckUsage{})

		mockRepository.EXPECT().
			FindOpponentDeckUsageStat(context.Background(), userId, gomock.Any(), gomock.Any(), deckId).
			Return(stat, nil)

		ret, err := usecase.GetOpponentDeckUsageStat(context.Background(), userId, yearMonth, environmentId, season, deckId)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})
}
