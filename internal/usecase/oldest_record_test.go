package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestOldestRecordUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockOldestRecordInterface(mockCtrl)
	usecase := NewOldestRecord(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockOldestRecordInterface,
		usecase OldestRecordInterface,
	){
		"GetOldestRecord": test_OldestRecordUsecase_GetOldestRecord,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_OldestRecordUsecase_GetOldestRecord(t *testing.T, mockRepository *mock_repository.MockOldestRecordInterface, usecase OldestRecordInterface) {
	t.Run("正常系_deck_id指定時はそのままrepositoryに渡される", func(t *testing.T) {
		userId := "user-01"
		deckId := "deck-01"
		eventDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

		record := entity.NewOldestRecord(&eventDate)

		mockRepository.EXPECT().
			FindOldestRecord(context.Background(), userId, deckId).
			Return(record, nil)

		ret, err := usecase.GetOldestRecord(context.Background(), userId, deckId)

		require.NoError(t, err)
		require.Equal(t, record, ret)
	})

	t.Run("正常系_該当記録が無い場合はEventDateがnilのまま返る", func(t *testing.T) {
		userId := "user-02"
		deckId := ""

		record := entity.NewOldestRecord(nil)

		mockRepository.EXPECT().
			FindOldestRecord(context.Background(), userId, deckId).
			Return(record, nil)

		ret, err := usecase.GetOldestRecord(context.Background(), userId, deckId)

		require.NoError(t, err)
		require.Nil(t, ret.EventDate)
	})
}
