package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func setup4TestKizunaUsecase(t *testing.T) (KizunaInterface, *mock_repository.MockKizunaInterface) {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockKizunaInterface(mockCtrl)

	return NewKizuna(mockRepository), mockRepository
}

func TestKizunaUsecase_GetKizuna(t *testing.T) {
	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_集計値からきずなLv.を算出して返す", func(t *testing.T) {
		usecase, mockRepository := setup4TestKizunaUsecase(t)

		mockRepository.EXPECT().FindKizunaDeckAggregates(context.Background(), userId).
			Return([]*entity.KizunaDeckAggregate{
				{
					DeckId:        "deck-01",
					EventDayCount: 18,
					StageCounts: map[entity.KizunaStageKind]int{
						entity.KizunaStageGymBattle:  20,
						entity.KizunaStageCityLeague: 4,
					},
					RecordCount:     24,
					MemoCount:       14,
					MemoTotalLength: 14 * 40,
					DeckCodeCount:   13,
					EveCodeCount:    4,
					MatchCount:      24,
					Wins:            8,
				},
			}, nil)

		kizuna, err := usecase.GetKizuna(context.Background(), userId)

		require.NoError(t, err)
		require.Equal(t, userId, kizuna.UserId)
		require.Len(t, kizuna.Decks, 1)
		// entity のテストと同じシナリオ。算出は entity に委譲されている
		require.Equal(t, 178, kizuna.Decks[0].Level)
	})

	t.Run("正常系_デッキが1つも無ければ空で返す", func(t *testing.T) {
		usecase, mockRepository := setup4TestKizunaUsecase(t)

		mockRepository.EXPECT().FindKizunaDeckAggregates(context.Background(), userId).
			Return([]*entity.KizunaDeckAggregate{}, nil)

		kizuna, err := usecase.GetKizuna(context.Background(), userId)

		require.NoError(t, err)
		require.Empty(t, kizuna.Decks)
	})

	t.Run("異常系_リポジトリのエラーはそのまま返す", func(t *testing.T) {
		usecase, mockRepository := setup4TestKizunaUsecase(t)

		wantErr := errors.New("db is down")
		mockRepository.EXPECT().FindKizunaDeckAggregates(context.Background(), userId).
			Return(nil, wantErr)

		kizuna, err := usecase.GetKizuna(context.Background(), userId)

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, kizuna)
	})
}
