package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"go.uber.org/mock/gomock"
)

func TestUserUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserInterface(mockCtrl)
	usecase := NewUser(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockUserInterface,
		usecase UserInterface,
	){
		"FindById": test_UserUsecase_FindById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_UserUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockUserInterface, usecase UserInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		user := &entity.User{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}
