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

const (
	DateLayout = "2006-01-02"
)

func TestOfficialEventUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockOfficialEventInterface(mockCtrl)
	usecase := NewOfficialEvent(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockOfficialEventInterface,
		usecase OfficialEventInterface,
	){
		"Find":     test_OfficialEventUsecase_Find,
		"FindById": test_OfficialEventUsecase_FindById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_OfficialEventUsecase_Find(
	t *testing.T,
	mockRepository *mock_repository.MockOfficialEventInterface,
	usecase OfficialEventInterface,
) {
	t.Run("正常系_#01", func(t *testing.T) {
		typeId := uint(1)
		leagueType := uint(0)
		startDate, _ := time.Parse(DateLayout, "2025-02-15T00:00:00Z")
		endDate, _ := time.Parse(DateLayout, "2025-02-15T00:00:00Z")

		officialEvents := []*entity.OfficialEvent{
			{
				ID: uint(606466),
			},
			{
				ID: uint(630879),
			},
		}

		mockRepository.EXPECT().Find(context.Background(), typeId, leagueType, startDate, endDate).Return(officialEvents, nil)

		ret, err := usecase.Find(context.Background(), typeId, leagueType, startDate, endDate)
		require.NoError(t, err)
		require.Equal(t, uint(606466), ret[0].ID)
		require.Equal(t, uint(630879), ret[1].ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
	})
}

func test_OfficialEventUsecase_FindById(
	t *testing.T,
	mockRepository *mock_repository.MockOfficialEventInterface,
	usecase OfficialEventInterface,
) {
	t.Run("正常系_#01", func(t *testing.T) {
		// https://players.pokemon-card.com/event_detail_search?event_holding_id=606466
		id := uint(606466)

		officialEvent := &entity.OfficialEvent{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(officialEvent, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		// https://players.pokemon-card.com/event_detail_search?event_holding_id=606466
		id := uint(606466)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}
