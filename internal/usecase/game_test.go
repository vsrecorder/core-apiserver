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

func TestGameUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockGameInterface(mockCtrl)
	usecase := NewGame(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockGameInterface,
		usecase GameInterface,
	){
		"FindById":      test_GameUsecase_FindById,
		"FindByMatchId": test_GameUsecase_FindByMatchId,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_GameUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockGameInterface, usecase GameInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		game := &entity.Game{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(game, nil)

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

func test_GameUsecase_FindByMatchId(t *testing.T, mockRepository *mock_repository.MockGameInterface, usecase GameInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		matchId, err := generateId()
		require.NoError(t, err)

		game := &entity.Game{
			ID:      id,
			MatchId: matchId,
		}

		games := []*entity.Game{
			game,
		}

		mockRepository.EXPECT().FindByMatchId(context.Background(), matchId).Return(games, nil)

		ret, err := usecase.FindByMatchId(context.Background(), matchId)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, matchId, ret[0].MatchId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		matchId, err := generateId()
		require.NoError(t, err)

		games := []*entity.Game{}

		mockRepository.EXPECT().FindByMatchId(context.Background(), matchId).Return(games, nil)

		ret, err := usecase.FindByMatchId(context.Background(), matchId)

		require.NoError(t, err)
		require.Equal(t, len(games), len(ret))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		matchId, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindByMatchId(context.Background(), matchId).Return(nil, errors.New(""))

		ret, err := usecase.FindByMatchId(context.Background(), matchId)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}
