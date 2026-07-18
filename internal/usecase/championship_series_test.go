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

func TestChampionshipSeriesUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)
	usecase := NewChampionshipSeries(mockRepository)

	series := &entity.ChampionshipSeries{ID: "series_2026"}

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_シリーズ一覧をそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().Find(context.Background()).Return([]*entity.ChampionshipSeries{series}, nil)

			ret, err := usecase.Find(context.Background())

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, series, ret[0])
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().Find(context.Background()).Return(nil, errors.New(""))

			ret, err := usecase.Find(context.Background())

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDのシリーズを返す", func(t *testing.T) {
			mockRepository.EXPECT().FindById(context.Background(), "series_2026").Return(series, nil)

			ret, err := usecase.FindById(context.Background(), "series_2026")

			require.NoError(t, err)
			require.Equal(t, series, ret)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().FindById(context.Background(), "series_2026").Return(nil, errors.New(""))

			ret, err := usecase.FindById(context.Background(), "series_2026")

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindByDate", func(t *testing.T) {
		date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local)

		t.Run("正常系_指定日が属するシリーズを返す", func(t *testing.T) {
			mockRepository.EXPECT().FindByDate(context.Background(), date).Return(series, nil)

			ret, err := usecase.FindByDate(context.Background(), date)

			require.NoError(t, err)
			require.Equal(t, series, ret)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().FindByDate(context.Background(), date).Return(nil, errors.New(""))

			ret, err := usecase.FindByDate(context.Background(), date)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})
}
