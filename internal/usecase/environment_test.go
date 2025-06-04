package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"go.uber.org/mock/gomock"
)

func TestEnvironmentUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	usecase := NewEnvironment(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockEnvironmentInterface,
		usecase EnvironmentInterface,
	){
		"Find":       test_EnvironmentUsecase_Find,
		"FindById":   test_EnvironmentUsecase_FindById,
		"FindByDate": test_EnvironmentUsecase_FindByDate,
		"FindByTerm": test_EnvironmentUsecase_FindByTerm,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_EnvironmentUsecase_Find(t *testing.T, mockRepository *mock_repository.MockEnvironmentInterface, usecase EnvironmentInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		environment := entity.NewEnvironment(
			id,
			title,
			fromDate,
			toDate,
		)

		environments := []*entity.Environment{
			environment,
		}

		mockRepository.EXPECT().Find(context.Background()).Return(environments, nil)

		ret, err := usecase.Find(context.Background())

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, title, ret[0].Title)
		require.Equal(t, fromDate, ret[0].FromDate)
		require.Equal(t, toDate, ret[0].ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		mockRepository.EXPECT().Find(context.Background()).Return(nil, errors.New(""))

		ret, err := usecase.Find(context.Background())

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_EnvironmentUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockEnvironmentInterface, usecase EnvironmentInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		environment := entity.NewEnvironment(
			id,
			title,
			fromDate,
			toDate,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(environment, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
		require.Equal(t, title, ret.Title)
		require.Equal(t, fromDate, ret.FromDate)
		require.Equal(t, toDate, ret.ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id := "sv11"
		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_EnvironmentUsecase_FindByDate(t *testing.T, mockRepository *mock_repository.MockEnvironmentInterface, usecase EnvironmentInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		environment := entity.NewEnvironment(
			id,
			title,
			fromDate,
			toDate,
		)

		date, _ := time.Parse(DateLayout, "2025-06-09")
		mockRepository.EXPECT().FindByDate(context.Background(), date).Return(environment, nil)

		ret, err := usecase.FindByDate(context.Background(), date)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
		require.Equal(t, title, ret.Title)
		require.Equal(t, fromDate, ret.FromDate)
		require.Equal(t, toDate, ret.ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		date, _ := time.Parse(DateLayout, "2025-06-09")
		mockRepository.EXPECT().FindByDate(context.Background(), date).Return(nil, errors.New(""))

		ret, err := usecase.FindByDate(context.Background(), date)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_EnvironmentUsecase_FindByTerm(t *testing.T, mockRepository *mock_repository.MockEnvironmentInterface, usecase EnvironmentInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		environment := entity.NewEnvironment(
			id,
			title,
			fromDate,
			toDate,
		)

		environments := []*entity.Environment{
			environment,
		}

		argFromDate, _ := time.Parse(DateLayout, "2025-06-09")
		argToDate, _ := time.Parse(DateLayout, "2025-06-09")
		mockRepository.EXPECT().FindByTerm(context.Background(), argFromDate, argToDate).Return(environments, nil)

		ret, err := usecase.FindByTerm(context.Background(), argFromDate, argToDate)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, title, ret[0].Title)
		require.Equal(t, fromDate, ret[0].FromDate)
		require.Equal(t, toDate, ret[0].ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		argFromDate, _ := time.Parse(DateLayout, "2025-06-09")
		argToDate, _ := time.Parse(DateLayout, "2025-06-09")
		mockRepository.EXPECT().FindByTerm(context.Background(), argFromDate, argToDate).Return(nil, errors.New(""))

		ret, err := usecase.FindByTerm(context.Background(), argFromDate, argToDate)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}
