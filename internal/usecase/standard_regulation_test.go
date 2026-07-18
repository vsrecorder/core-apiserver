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

func TestStandardRegulationUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockStandardRegulationInterface(mockCtrl)
	usecase := NewStandardRegulation(mockRepository)

	regulation := &entity.StandardRegulation{ID: "regulation-g"}

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_レギュレーション一覧をそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().Find(context.Background()).Return([]*entity.StandardRegulation{regulation}, nil)

			ret, err := usecase.Find(context.Background())

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, regulation, ret[0])
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().Find(context.Background()).Return(nil, errors.New(""))

			ret, err := usecase.Find(context.Background())

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDのレギュレーションを返す", func(t *testing.T) {
			mockRepository.EXPECT().FindById(context.Background(), "regulation-g").Return(regulation, nil)

			ret, err := usecase.FindById(context.Background(), "regulation-g")

			require.NoError(t, err)
			require.Equal(t, regulation, ret)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().FindById(context.Background(), "regulation-g").Return(nil, errors.New(""))

			ret, err := usecase.FindById(context.Background(), "regulation-g")

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindByDate", func(t *testing.T) {
		date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local)

		t.Run("正常系_指定日のレギュレーションを返す", func(t *testing.T) {
			mockRepository.EXPECT().FindByDate(context.Background(), date).Return(regulation, nil)

			ret, err := usecase.FindByDate(context.Background(), date)

			require.NoError(t, err)
			require.Equal(t, regulation, ret)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository.EXPECT().FindByDate(context.Background(), date).Return(nil, errors.New(""))

			ret, err := usecase.FindByDate(context.Background(), date)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})
}
