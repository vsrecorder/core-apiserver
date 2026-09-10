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

func TestOpponentDeckUsageStatUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockOpponentDeckUsageStatInterface(mockCtrl)
	mockEnvironmentRepository := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	mockOfficialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)
	// 環境の例外イベント(official_event_environments)そのものの検証は
	// environment_scope_test.go で行う。ここでは期間の組み立てだけを見たいので、
	// 環境指定時に引かれても例外なし(nil)を返す。
	mockOfficialEventEnvironmentRepo.EXPECT().FindAll(gomock.Any()).Return(nil, nil).AnyTimes()
	mockStandardRegulationRepository := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
	mockChampionshipSeriesRepository := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)
	usecase := NewOpponentDeckUsageStat(mockRepository, mockEnvironmentRepository, mockOfficialEventEnvironmentRepo, mockStandardRegulationRepository, mockChampionshipSeriesRepository)

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
	t.Run("正常系_week指定時はその週(月曜0時〜翌月曜0時)の期間で集計する", func(t *testing.T) {
		userId := "user-00"

		stat := entity.NewOpponentDeckUsageStat(userId, 3, []*entity.OpponentDeckUsage{})

		fromDate := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)

		mockRepository.EXPECT().
			FindOpponentDeckUsageStat(context.Background(), userId, statPeriodOf(fromDate, toDate), "", uint(0), false).
			Return(stat, nil)

		ret, err := usecase.GetOpponentDeckUsageStat(context.Background(), userId, "2026-08-19", "", "", "", "", 0, "", false)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_deck_id指定時はそのままrepositoryに渡される", func(t *testing.T) {
		userId := "user-01"
		yearMonth := "2026-06"
		environmentId := ""
		season := ""
		standardRegulationId := ""
		deckId := "deck-01"

		stat := entity.NewOpponentDeckUsageStat(userId, 5, []*entity.OpponentDeckUsage{})

		mockRepository.EXPECT().
			FindOpponentDeckUsageStat(context.Background(), userId, gomock.Any(), deckId, uint(0), false).
			Return(stat, nil)

		ret, err := usecase.GetOpponentDeckUsageStat(context.Background(), userId, "", yearMonth, environmentId, season, standardRegulationId, 0, deckId, false)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_deck_id未指定でも空文字のまま渡される", func(t *testing.T) {
		userId := "user-02"
		yearMonth := "2026-06"
		environmentId := ""
		season := ""
		standardRegulationId := ""
		deckId := ""

		stat := entity.NewOpponentDeckUsageStat(userId, 0, []*entity.OpponentDeckUsage{})

		mockRepository.EXPECT().
			FindOpponentDeckUsageStat(context.Background(), userId, gomock.Any(), deckId, uint(0), false).
			Return(stat, nil)

		ret, err := usecase.GetOpponentDeckUsageStat(context.Background(), userId, "", yearMonth, environmentId, season, standardRegulationId, 0, deckId, false)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	// 「全期間」フィルタはyear_month等のクエリパラメータを一切送らないため、
	// この場合に当月だけへ絞り込んでしまう不具合の再発防止テスト。
	// fromDate/toDateがゼロ値のままrepositoryに渡され、event_dateによる絞り込みが行われないことを確認する。
	t.Run("正常系_フィルタ未指定時はfromDate_toDateがゼロ値のまま渡される(全期間)", func(t *testing.T) {
		userId := "user-03"
		yearMonth := ""
		environmentId := ""
		season := ""
		standardRegulationId := ""
		deckId := "deck-01"

		stat := entity.NewOpponentDeckUsageStat(userId, 3, []*entity.OpponentDeckUsage{})

		mockRepository.EXPECT().
			FindOpponentDeckUsageStat(context.Background(), userId, repository.StatPeriod{}, deckId, uint(0), false).
			Return(stat, nil)

		ret, err := usecase.GetOpponentDeckUsageStat(context.Background(), userId, "", yearMonth, environmentId, season, standardRegulationId, 0, deckId, false)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})
}
