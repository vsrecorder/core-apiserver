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

func setup4UserStatUsecase(t *testing.T) (
	*mock_repository.MockUserStatInterface,
	*mock_repository.MockEnvironmentInterface,
	*mock_repository.MockStandardRegulationInterface,
	*mock_repository.MockChampionshipSeriesInterface,
	UserStatInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockUserStatRepo := mock_repository.NewMockUserStatInterface(mockCtrl)
	mockEnvironmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	mockRegulationRepo := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
	mockSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	usecase := NewUserStat(mockUserStatRepo, mockEnvironmentRepo, mockRegulationRepo, mockSeriesRepo)

	return mockUserStatRepo, mockEnvironmentRepo, mockRegulationRepo, mockSeriesRepo, usecase
}

func TestUserStatUsecase_GetUserStat(t *testing.T) {
	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	stat := &entity.UserStat{}

	t.Run("正常系_year_month指定時はその月の期間で集計する", func(t *testing.T) {
		mockUserStatRepo, _, _, _, usecase := setup4UserStatUsecase(t)

		fromDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)

		mockUserStatRepo.EXPECT().FindUserStat(context.Background(), userId, fromDate, toDate).Return(stat, nil)

		ret, err := usecase.GetUserStat(context.Background(), userId, "2026-06", "", "", "")

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_すべて未指定なら当月の期間で集計する", func(t *testing.T) {
		mockUserStatRepo, _, _, _, usecase := setup4UserStatUsecase(t)

		// 月末の23:59でも「当月」の期間になることを固定時刻で検証する
		overrideTimeNow(t, time.Date(2026, 7, 31, 23, 59, 59, 0, time.Local))

		fromDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)

		mockUserStatRepo.EXPECT().FindUserStat(context.Background(), userId, fromDate, toDate).Return(stat, nil)

		ret, err := usecase.GetUserStat(context.Background(), userId, "", "", "", "")

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_season指定時はチャンピオンシップシリーズの期間で集計する", func(t *testing.T) {
		mockUserStatRepo, _, _, mockSeriesRepo, usecase := setup4UserStatUsecase(t)

		cs := &entity.ChampionshipSeries{
			ID:       "series_2026",
			FromDate: time.Date(2025, 7, 1, 0, 0, 0, 0, time.Local),
			ToDate:   time.Date(2026, 6, 30, 0, 0, 0, 0, time.Local),
		}

		// seasonにはchampionship_series.idの接頭辞を除いた識別子が渡される
		mockSeriesRepo.EXPECT().FindById(context.Background(), "series_2026").Return(cs, nil)

		fromDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local) // to_dateの翌日0時がexclusive上限

		mockUserStatRepo.EXPECT().FindUserStat(context.Background(), userId, fromDate, toDate).Return(stat, nil)

		ret, err := usecase.GetUserStat(context.Background(), userId, "", "", "2026", "")

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_environment指定時は環境の期間で集計する", func(t *testing.T) {
		mockUserStatRepo, mockEnvironmentRepo, _, _, usecase := setup4UserStatUsecase(t)

		env := entity.NewEnvironment(
			"sv11",
			"ブラックボルト/ホワイトフレア",
			time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
			time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local),
		)

		mockEnvironmentRepo.EXPECT().FindById(context.Background(), "sv11").Return(env, nil)

		fromDate := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local) // to_dateの翌日0時がexclusive上限

		mockUserStatRepo.EXPECT().FindUserStat(context.Background(), userId, fromDate, toDate).Return(stat, nil)

		ret, err := usecase.GetUserStat(context.Background(), userId, "", "sv11", "", "")

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_year_monthとenvironmentの両指定時は期間の交差を取る", func(t *testing.T) {
		mockUserStatRepo, mockEnvironmentRepo, _, _, usecase := setup4UserStatUsecase(t)

		// 環境期間: 2026-06-06〜2026-07-31、year_month: 2026-06
		env := entity.NewEnvironment(
			"sv11",
			"ブラックボルト/ホワイトフレア",
			time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local),
			time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local),
		)

		mockEnvironmentRepo.EXPECT().FindById(context.Background(), "sv11").Return(env, nil)

		// 交差: from=環境の開始(6/6)、to=月末翌日(7/1)
		fromDate := time.Date(2026, 6, 6, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)

		mockUserStatRepo.EXPECT().FindUserStat(context.Background(), userId, fromDate, toDate).Return(stat, nil)

		ret, err := usecase.GetUserStat(context.Background(), userId, "2026-06", "sv11", "", "")

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_regulation指定時はレギュレーションの期間で集計する", func(t *testing.T) {
		mockUserStatRepo, _, mockRegulationRepo, _, usecase := setup4UserStatUsecase(t)

		reg := &entity.StandardRegulation{
			ID:       "regulation-g",
			FromDate: time.Date(2026, 1, 24, 0, 0, 0, 0, time.Local),
			ToDate:   time.Date(2027, 1, 22, 0, 0, 0, 0, time.Local),
		}

		mockRegulationRepo.EXPECT().FindById(context.Background(), "regulation-g").Return(reg, nil)

		fromDate := time.Date(2026, 1, 24, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2027, 1, 23, 0, 0, 0, 0, time.Local) // to_dateの翌日0時がexclusive上限

		mockUserStatRepo.EXPECT().FindUserStat(context.Background(), userId, fromDate, toDate).Return(stat, nil)

		ret, err := usecase.GetUserStat(context.Background(), userId, "", "", "", "regulation-g")

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("異常系_year_monthの形式が不正ならエラーを返す", func(t *testing.T) {
		_, _, _, _, usecase := setup4UserStatUsecase(t)

		ret, err := usecase.GetUserStat(context.Background(), userId, "202606", "", "", "")

		require.Error(t, err)
		require.Nil(t, ret)
	})

	t.Run("異常系_環境の取得エラーをそのまま返す", func(t *testing.T) {
		_, mockEnvironmentRepo, _, _, usecase := setup4UserStatUsecase(t)

		mockEnvironmentRepo.EXPECT().FindById(context.Background(), "sv11").Return(nil, errors.New(""))

		ret, err := usecase.GetUserStat(context.Background(), userId, "", "sv11", "", "")

		require.Error(t, err)
		require.Nil(t, ret)
	})

	t.Run("異常系_レギュレーションの取得エラーをそのまま返す", func(t *testing.T) {
		_, _, mockRegulationRepo, _, usecase := setup4UserStatUsecase(t)

		mockRegulationRepo.EXPECT().FindById(context.Background(), "regulation-g").Return(nil, errors.New(""))

		ret, err := usecase.GetUserStat(context.Background(), userId, "", "", "", "regulation-g")

		require.Error(t, err)
		require.Nil(t, ret)
	})

	t.Run("異常系_シーズンの取得エラーをそのまま返す", func(t *testing.T) {
		_, _, _, mockSeriesRepo, usecase := setup4UserStatUsecase(t)

		mockSeriesRepo.EXPECT().FindById(context.Background(), "series_2026").Return(nil, errors.New(""))

		ret, err := usecase.GetUserStat(context.Background(), userId, "", "", "2026", "")

		require.Error(t, err)
		require.Nil(t, ret)
	})

	t.Run("異常系_集計リポジトリのエラーをそのまま返す", func(t *testing.T) {
		mockUserStatRepo, _, _, _, usecase := setup4UserStatUsecase(t)

		mockUserStatRepo.EXPECT().FindUserStat(context.Background(), userId, gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		ret, err := usecase.GetUserStat(context.Background(), userId, "2026-06", "", "", "")

		require.Error(t, err)
		require.Nil(t, ret)
	})
}
