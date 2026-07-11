package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestDeckUsageStatUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckUsageStatInterface(mockCtrl)
	mockEnvironmentRepository := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	mockStandardRegulationRepository := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
	mockChampionshipSeriesRepository := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)
	usecase := NewDeckUsageStat(mockRepository, mockEnvironmentRepository, mockStandardRegulationRepository, mockChampionshipSeriesRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockDeckUsageStatInterface,
		usecase DeckUsageStatInterface,
	){
		"AllTime_期間条件を一切付けずにrepositoryへ委譲する": test_DeckUsageStatUsecase_AllTime,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_DeckUsageStatUsecase_AllTime(t *testing.T, mockRepository *mock_repository.MockDeckUsageStatInterface, usecase DeckUsageStatInterface) {
	userId := "user-01"

	want := entity.NewDeckUsageStat(userId, 0, []*entity.DeckUsage{})

	mockRepository.EXPECT().
		FindDeckUsageStat(gomock.Any(), userId, time.Time{}, time.Time{}).
		Return(want, nil)

	// year_month/season/regulation_idを指定していても all_time=true の場合は無視され、
	// 期間条件なしでrepositoryが呼ばれる。
	got, err := usecase.GetDeckUsageStat(context.Background(), userId, "2026-06", "", "spring", "", true)

	require.NoError(t, err)
	require.Equal(t, want, got)
}
