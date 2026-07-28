package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestCurrentSeasonLabel(t *testing.T) {
	t.Run("正常系_championship_seriesでnowを含む行のIDからseason識別子を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.Local)
		cs := entity.NewChampionshipSeries(
			"series_2027", "チャンピオンシップシリーズ2027",
			time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2027, 8, 31, 0, 0, 0, 0, time.Local),
		)
		repo.EXPECT().FindByDate(gomock.Any(), now).Return(cs, nil)

		label, err := CurrentSeasonLabel(t.Context(), repo, now)

		require.NoError(t, err)
		require.Equal(t, "2027", label)
	})

	t.Run("異常系_該当するシーズンがchampionship_seriesに無ければエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.Local)
		repo.EXPECT().FindByDate(gomock.Any(), now).Return(nil, apperror.ErrRecordNotFound)

		_, err := CurrentSeasonLabel(t.Context(), repo, now)

		require.Error(t, err)
	})
}

func TestSeasonRange(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)

	t.Run("正常系_season空文字ならFindByDateでnowが属する現在のシーズンの期間を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		cs := entity.NewChampionshipSeries(
			"series_2026", "チャンピオンシップシリーズ2026",
			time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local),
		)
		repo.EXPECT().FindByDate(gomock.Any(), now).Return(cs, nil)

		fromDate, toDate, err := seasonRange(t.Context(), repo, "", now)

		require.NoError(t, err)
		require.Equal(t, time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("正常系_season指定時はFindByIdで series_+season の期間を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		cs := entity.NewChampionshipSeries(
			"series_2024", "チャンピオンシップシリーズ2024",
			time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2024, 8, 31, 0, 0, 0, 0, time.Local),
		)
		repo.EXPECT().FindById(gomock.Any(), "series_2024").Return(cs, nil)

		fromDate, toDate, err := seasonRange(t.Context(), repo, "2024", now)

		require.NoError(t, err)
		require.Equal(t, time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("異常系_championship_seriesに該当行が無ければエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		repo.EXPECT().FindById(gomock.Any(), "series_not-a-year").Return(nil, apperror.ErrRecordNotFound)

		_, _, err := seasonRange(t.Context(), repo, "not-a-year", now)

		require.Error(t, err)
	})
}

func TestPreviousSeasonRange(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)

	t.Run("正常系_season空文字なら現在のシーズンのひとつ前の期間を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		current := entity.NewChampionshipSeries(
			"series_2026", "チャンピオンシップシリーズ2026",
			time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local),
		)
		previous := entity.NewChampionshipSeries(
			"series_2025", "チャンピオンシップシリーズ2025",
			time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2025, 8, 31, 0, 0, 0, 0, time.Local),
		)

		repo.EXPECT().FindByDate(gomock.Any(), now).Return(current, nil)
		repo.EXPECT().FindByDate(gomock.Any(), time.Date(2025, 8, 31, 0, 0, 0, 0, time.Local)).Return(previous, nil)

		fromDate, toDate, err := previousSeasonRange(t.Context(), repo, "", now)

		require.NoError(t, err)
		require.Equal(t, time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("正常系_season指定時はその年のひとつ前のシーズン期間を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		current := entity.NewChampionshipSeries(
			"series_2024", "チャンピオンシップシリーズ2024",
			time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2024, 8, 31, 0, 0, 0, 0, time.Local),
		)
		previous := entity.NewChampionshipSeries(
			"series_2023", "チャンピオンシップシリーズ2023",
			time.Date(2022, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2023, 8, 31, 0, 0, 0, 0, time.Local),
		)

		repo.EXPECT().FindById(gomock.Any(), "series_2024").Return(current, nil)
		repo.EXPECT().FindByDate(gomock.Any(), time.Date(2023, 8, 31, 0, 0, 0, 0, time.Local)).Return(previous, nil)

		fromDate, toDate, err := previousSeasonRange(t.Context(), repo, "2024", now)

		require.NoError(t, err)
		require.Equal(t, time.Date(2022, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("異常系_championship_seriesに該当行が無ければエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		repo.EXPECT().FindById(gomock.Any(), "series_not-a-year").Return(nil, apperror.ErrRecordNotFound)

		_, _, err := previousSeasonRange(t.Context(), repo, "not-a-year", now)

		require.Error(t, err)
	})
}

func TestPeriodDateRange(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)

	t.Run("正常系_いずれも未指定ならゼロ値を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		standardRegulationRepo := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
		championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		fromDate, toDate, err := PeriodDateRange(
			t.Context(), environmentRepo, standardRegulationRepo, championshipSeriesRepo,
			"", "", "", now,
		)

		require.NoError(t, err)
		require.True(t, fromDate.IsZero())
		require.True(t, toDate.IsZero())
	})

	t.Run("正常系_environmentIdのみ指定でその環境の期間を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		standardRegulationRepo := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
		championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		env := entity.NewEnvironment(
			"sv9a", "熱風のアリーナ",
			time.Date(2025, 3, 14, 0, 0, 0, 0, time.Local),
			time.Date(2025, 4, 17, 0, 0, 0, 0, time.Local),
		)
		environmentRepo.EXPECT().FindById(gomock.Any(), "sv9a").Return(env, nil)

		fromDate, toDate, err := PeriodDateRange(
			t.Context(), environmentRepo, standardRegulationRepo, championshipSeriesRepo,
			"sv9a", "", "", now,
		)

		require.NoError(t, err)
		require.Equal(t, time.Date(2025, 3, 14, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2025, 4, 18, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("正常系_season指定時はseasonRangeの期間を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		standardRegulationRepo := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
		championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		cs := entity.NewChampionshipSeries(
			"series_2024", "チャンピオンシップシリーズ2024",
			time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2024, 8, 31, 0, 0, 0, 0, time.Local),
		)
		championshipSeriesRepo.EXPECT().FindById(gomock.Any(), "series_2024").Return(cs, nil)

		fromDate, toDate, err := PeriodDateRange(
			t.Context(), environmentRepo, standardRegulationRepo, championshipSeriesRepo,
			"", "2024", "", now,
		)

		require.NoError(t, err)
		require.Equal(t, time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("正常系_season+regulationは期間の交差を取る", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		standardRegulationRepo := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
		championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		cs := entity.NewChampionshipSeries(
			"series_2026", "チャンピオンシップシリーズ2026",
			time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local),
		)
		championshipSeriesRepo.EXPECT().FindById(gomock.Any(), "series_2026").Return(cs, nil)

		reg := entity.NewStandardRegulation(
			"GHI", "G・H・I",
			time.Date(2025, 1, 24, 0, 0, 0, 0, time.Local),
			time.Date(2025, 12, 18, 0, 0, 0, 0, time.Local),
		)
		standardRegulationRepo.EXPECT().FindById(gomock.Any(), "GHI").Return(reg, nil)

		fromDate, toDate, err := PeriodDateRange(
			t.Context(), environmentRepo, standardRegulationRepo, championshipSeriesRepo,
			"", "2026", "GHI", now,
		)

		require.NoError(t, err)
		// season(2025-09-01〜2026-09-01)とregulation(2025-01-24〜2025-12-19)の交差
		require.Equal(t, time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), fromDate)
		require.Equal(t, time.Date(2025, 12, 19, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("異常系_environmentIdが見つからなければエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		standardRegulationRepo := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
		championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		environmentRepo.EXPECT().FindById(gomock.Any(), "unknown").Return(nil, apperror.ErrRecordNotFound)

		_, _, err := PeriodDateRange(
			t.Context(), environmentRepo, standardRegulationRepo, championshipSeriesRepo,
			"unknown", "", "", now,
		)

		require.Error(t, err)
	})

	t.Run("異常系_regulationIdが見つからなければエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		standardRegulationRepo := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
		championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

		standardRegulationRepo.EXPECT().FindById(gomock.Any(), "unknown").Return(nil, apperror.ErrRecordNotFound)

		_, _, err := PeriodDateRange(
			t.Context(), environmentRepo, standardRegulationRepo, championshipSeriesRepo,
			"", "", "unknown", now,
		)

		require.Error(t, err)
	})
}
