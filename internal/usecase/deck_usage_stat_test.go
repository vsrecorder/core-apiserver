package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestDeckUsageStatUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckUsageStatInterface(mockCtrl)
	mockEnvironmentRepository := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	mockOfficialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)
	// 環境の例外イベント(official_event_environments)そのものの検証は
	// environment_scope_test.go で行う。ここでは期間の組み立てだけを見たいので、
	// 環境指定時に引かれても例外なし(nil)を返す。
	mockOfficialEventEnvironmentRepo.EXPECT().FindAll(gomock.Any()).Return(nil, nil).AnyTimes()
	mockStandardRegulationRepository := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
	mockChampionshipSeriesRepository := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)
	usecase := NewDeckUsageStat(mockRepository, mockEnvironmentRepository, mockOfficialEventEnvironmentRepo, mockStandardRegulationRepository, mockChampionshipSeriesRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockDeckUsageStatInterface,
		usecase DeckUsageStatInterface,
	){
		"AllTime_期間条件を一切付けずにrepositoryへ委譲する":          test_DeckUsageStatUsecase_AllTime,
		"Week_週指定時はその週の期間でrepositoryへ委譲する":            test_DeckUsageStatUsecase_Week,
		"ExcludeDefaultMatches_不戦の除外指定をrepositoryへ渡す": test_DeckUsageStatUsecase_ExcludeDefaultMatches,
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
		FindDeckUsageStat(gomock.Any(), userId, repository.StatPeriod{}, uint(0), false).
		Return(want, nil)

	// year_month/season/regulation_idを指定していても all_time=true の場合は無視され、
	// 期間条件なしでrepositoryが呼ばれる。
	got, err := usecase.GetDeckUsageStat(context.Background(), userId, "", "2026-06", "", "spring", "", 0, true, false)

	require.NoError(t, err)
	require.Equal(t, want, got)
}

func test_DeckUsageStatUsecase_Week(t *testing.T, mockRepository *mock_repository.MockDeckUsageStatInterface, usecase DeckUsageStatInterface) {
	userId := "user-01"

	want := entity.NewDeckUsageStat(userId, 0, []*entity.DeckUsage{})

	// 週内の任意日(日曜)を渡しても、その週の月曜0時〜翌月曜0時に正規化される
	fromDate := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)

	mockRepository.EXPECT().
		FindDeckUsageStat(gomock.Any(), userId, statPeriodOf(fromDate, toDate), uint(0), false).
		Return(want, nil)

	// year_month も同時に指定しているが week が優先される
	got, err := usecase.GetDeckUsageStat(context.Background(), userId, "2026-08-23", "2026-06", "", "", "", 0, false, false)

	require.NoError(t, err)
	require.Equal(t, want, got)
}

// 不戦勝/不戦敗の除外指定は期間の組み立てとは無関係に、そのまま repository へ渡す。
func test_DeckUsageStatUsecase_ExcludeDefaultMatches(t *testing.T, mockRepository *mock_repository.MockDeckUsageStatInterface, usecase DeckUsageStatInterface) {
	userId := "user-01"

	want := entity.NewDeckUsageStat(userId, 0, []*entity.DeckUsage{})

	mockRepository.EXPECT().
		FindDeckUsageStat(gomock.Any(), userId, repository.StatPeriod{}, uint(0), true).
		Return(want, nil)

	got, err := usecase.GetDeckUsageStat(context.Background(), userId, "", "", "", "", "", 0, true, true)

	require.NoError(t, err)
	require.Equal(t, want, got)
}
