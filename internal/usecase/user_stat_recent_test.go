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

func TestUserStatRecentUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserStatRecentInterface(mockCtrl)
	mockEnvironmentRepository := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	usecase := NewUserStatRecent(mockRepository, mockEnvironmentRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockUserStatRecentInterface,
		mockEnvironmentRepository *mock_repository.MockEnvironmentInterface,
		usecase UserStatRecentInterface,
	){
		"GetRecentMatches": test_UserStatRecentUsecase_GetRecentMatches,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, mockEnvironmentRepository, usecase)
		})
	}
}

func test_UserStatRecentUsecase_GetRecentMatches(
	t *testing.T,
	mockRepository *mock_repository.MockUserStatRecentInterface,
	mockEnvironmentRepository *mock_repository.MockEnvironmentInterface,
	usecase UserStatRecentInterface,
) {
	t.Run("正常系_#01_表示件数の半分をウィンドウ幅としたローリング勝率が計算される", func(t *testing.T) {
		userId := "user-01"
		count := 4
		deckId := ""
		// windowSize = count/2 = 2 のため、表示件数(4件)より前情報として1件（windowSize-1）多く取得する
		fetchCount := 5

		now := time.Now()
		date0 := now.AddDate(0, 0, -5) // 前情報（表示対象外）
		date1 := now.AddDate(0, 0, -4)
		date2 := now.AddDate(0, 0, -3)
		date3 := now.AddDate(0, 0, -2)
		date4 := now.AddDate(0, 0, -1)

		rawMatches := []*entity.RecentMatch{
			entity.NewRecentMatch(0, date0, "deck-01", "対戦相手デッキA", true, 0, "", "", nil),  // 前情報: 勝ち
			entity.NewRecentMatch(0, date1, "deck-01", "対戦相手デッキB", false, 0, "", "", nil), // 表示1戦目: 負け
			entity.NewRecentMatch(0, date2, "deck-01", "対戦相手デッキA", true, 0, "", "", nil),  // 表示2戦目: 勝ち
			entity.NewRecentMatch(0, date3, "deck-01", "対戦相手デッキA", true, 0, "", "", nil),  // 表示3戦目: 勝ち
			entity.NewRecentMatch(0, date4, "deck-01", "対戦相手デッキB", false, 0, "", "", nil), // 表示4戦目: 負け
		}

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId).Return(rawMatches, nil)
		mockEnvironmentRepository.EXPECT().FindByTerm(context.Background(), date0, date4).Return(nil, nil)

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId)

		require.NoError(t, err)
		require.Equal(t, count, ret.Count)
		require.Equal(t, 4, ret.TotalMatches)
		require.Equal(t, 2, ret.Wins)
		require.InDelta(t, 0.5, ret.WinRate, 0.0001)
		require.Len(t, ret.Matches, 4)

		// 表示1戦目は単独では負け(0%)だが、前情報の1戦(勝ち)を含むウィンドウで計算されるため0%にはならない
		require.Equal(t, 1, ret.Matches[0].Sequence)
		require.False(t, ret.Matches[0].VictoryFlg)
		require.InDelta(t, 0.5, ret.Matches[0].RollingWinRate, 0.0001)

		require.Equal(t, 2, ret.Matches[1].Sequence)
		require.InDelta(t, 0.5, ret.Matches[1].RollingWinRate, 0.0001)

		require.Equal(t, 3, ret.Matches[2].Sequence)
		require.InDelta(t, 1.0, ret.Matches[2].RollingWinRate, 0.0001)

		// 表示4戦目は単独では負け(0%)だが、直前の表示3戦目(勝ち)を含むウィンドウで計算されるため0%にはならない
		require.Equal(t, 4, ret.Matches[3].Sequence)
		require.False(t, ret.Matches[3].VictoryFlg)
		require.InDelta(t, 0.5, ret.Matches[3].RollingWinRate, 0.0001)
	})

	t.Run("正常系_#02_実際の対戦数がcountに満たない場合は取得できた分だけ表示する", func(t *testing.T) {
		userId := "user-02"
		count := 4
		deckId := ""
		fetchCount := 5

		now := time.Now()
		date1 := now.AddDate(0, 0, -2)
		date2 := now.AddDate(0, 0, -1)

		// 対戦記録が2件しかなく、fetchCount(5件)に満たない
		rawMatches := []*entity.RecentMatch{
			entity.NewRecentMatch(0, date1, "deck-01", "対戦相手デッキA", true, 0, "", "", nil),
			entity.NewRecentMatch(0, date2, "deck-01", "対戦相手デッキB", false, 0, "", "", nil),
		}

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId).Return(rawMatches, nil)
		mockEnvironmentRepository.EXPECT().FindByTerm(context.Background(), date1, date2).Return(nil, nil)

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId)

		require.NoError(t, err)
		require.Equal(t, 2, ret.TotalMatches)
		require.Len(t, ret.Matches, 2)
		require.Equal(t, 1, ret.Matches[0].Sequence)
		require.Equal(t, 2, ret.Matches[1].Sequence)
	})

	t.Run("正常系_#03_試合が0件の場合は空配列と勝率0を返す", func(t *testing.T) {
		userId := "user-03"
		count := 10
		deckId := "deck-99"
		fetchCount := 14 // windowSize = 10/2 = 5 → count + windowSize - 1

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId).Return([]*entity.RecentMatch{}, nil)

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId)

		require.NoError(t, err)
		require.Equal(t, 0, ret.TotalMatches)
		require.Equal(t, 0.0, ret.WinRate)
		require.Empty(t, ret.Matches)
	})

	t.Run("異常系_#01_repositoryがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
		userId := "user-04"
		count := 10
		deckId := ""
		fetchCount := 14

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId).Return(nil, errors.New("db error"))

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId)

		require.Error(t, err)
		require.Nil(t, ret)
	})
}
