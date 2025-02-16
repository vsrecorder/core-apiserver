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

func TestTonamelEventUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockTonamelEventInterface(mockCtrl)
	usecase := NewTonamelEvent(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockTonamelEventInterface,
		usecase *TonamelEvent,
	){
		"FindById": test_TonamelEventUsecase_FindById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_TonamelEventUsecase_FindById(
	t *testing.T,
	mockRepository *mock_repository.MockTonamelEventInterface,
	usecase *TonamelEvent,
) {
	t.Run("正常系_#01", func(t *testing.T) {
		id := "61ozP"

		tonamelEvent := &entity.TonamelEvent{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(tonamelEvent, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id := "61ozP"

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}
