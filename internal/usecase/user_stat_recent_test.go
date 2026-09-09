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
	// 環境の例外(official_event_environments)の検証は
	// test_UserStatRecentUsecase_GetRecentMatches_EnvironmentOverride で行う。
	// ここでは例外なし(空)を返す。
	mockOfficialEventEnvironmentRepository := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)
	mockOfficialEventEnvironmentRepository.EXPECT().FindAll(gomock.Any()).Return(nil, nil).AnyTimes()
	usecase := NewUserStatRecent(mockRepository, mockEnvironmentRepository, mockOfficialEventEnvironmentRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockUserStatRecentInterface,
		mockEnvironmentRepository *mock_repository.MockEnvironmentInterface,
		usecase UserStatRecentInterface,
	){
		"GetRecentMatches":              test_UserStatRecentUsecase_GetRecentMatches,
		"GetRecentMatches_DrawExcluded": test_UserStatRecentUsecase_GetRecentMatches_DrawExcluded,
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
	t.Run("正常系_表示件数の半分をウィンドウ幅としたローリング勝率が計算される", func(t *testing.T) {
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
			entity.NewRecentMatch(0, date0, 0, "deck-01", "対戦相手デッキA", true, false, 0, "", "", nil),  // 前情報: 勝ち
			entity.NewRecentMatch(0, date1, 0, "deck-01", "対戦相手デッキB", false, false, 0, "", "", nil), // 表示1戦目: 負け
			entity.NewRecentMatch(0, date2, 0, "deck-01", "対戦相手デッキA", true, false, 0, "", "", nil),  // 表示2戦目: 勝ち
			entity.NewRecentMatch(0, date3, 0, "deck-01", "対戦相手デッキA", true, false, 0, "", "", nil),  // 表示3戦目: 勝ち
			entity.NewRecentMatch(0, date4, 0, "deck-01", "対戦相手デッキB", false, false, 0, "", "", nil), // 表示4戦目: 負け
		}

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId, uint(0)).Return(rawMatches, nil)
		mockEnvironmentRepository.EXPECT().FindByTerm(context.Background(), date0, date4).Return(nil, nil)

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId, 0)

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

	t.Run("正常系_実際の対戦数がcountに満たない場合は取得できた分だけ表示する", func(t *testing.T) {
		userId := "user-02"
		count := 4
		deckId := ""
		fetchCount := 5

		now := time.Now()
		date1 := now.AddDate(0, 0, -2)
		date2 := now.AddDate(0, 0, -1)

		// 対戦記録が2件しかなく、fetchCount(5件)に満たない
		rawMatches := []*entity.RecentMatch{
			entity.NewRecentMatch(0, date1, 0, "deck-01", "対戦相手デッキA", true, false, 0, "", "", nil),
			entity.NewRecentMatch(0, date2, 0, "deck-01", "対戦相手デッキB", false, false, 0, "", "", nil),
		}

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId, uint(0)).Return(rawMatches, nil)
		mockEnvironmentRepository.EXPECT().FindByTerm(context.Background(), date1, date2).Return(nil, nil)

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId, 0)

		require.NoError(t, err)
		require.Equal(t, 2, ret.TotalMatches)
		require.Len(t, ret.Matches, 2)
		require.Equal(t, 1, ret.Matches[0].Sequence)
		require.Equal(t, 2, ret.Matches[1].Sequence)
	})

	t.Run("正常系_試合が0件の場合は空配列と勝率0を返す", func(t *testing.T) {
		userId := "user-03"
		count := 10
		deckId := "deck-99"
		fetchCount := 14 // windowSize = 10/2 = 5 → count + windowSize - 1

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId, uint(0)).Return([]*entity.RecentMatch{}, nil)

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId, 0)

		require.NoError(t, err)
		require.Equal(t, 0, ret.TotalMatches)
		require.Equal(t, 0.0, ret.WinRate)
		require.Empty(t, ret.Matches)
	})

	t.Run("異常系_repositoryがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
		userId := "user-04"
		count := 10
		deckId := ""
		fetchCount := 14

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId, uint(0)).Return(nil, errors.New("db error"))

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId, 0)

		require.Error(t, err)
		require.Nil(t, ret)
	})
}

// 引き分けが勝率(全体・ローリング)の分母から除外されることを検証する。
func test_UserStatRecentUsecase_GetRecentMatches_DrawExcluded(
	t *testing.T,
	mockRepository *mock_repository.MockUserStatRecentInterface,
	mockEnvironmentRepository *mock_repository.MockEnvironmentInterface,
	usecase UserStatRecentInterface,
) {
	t.Run("正常系_引き分けは勝率の分母から除外される", func(t *testing.T) {
		userId := "user-draw"
		count := 4
		deckId := ""
		fetchCount := 5 // windowSize = count/2 = 2

		now := time.Now()
		date0 := now.AddDate(0, 0, -5)
		date1 := now.AddDate(0, 0, -4)
		date2 := now.AddDate(0, 0, -3)
		date3 := now.AddDate(0, 0, -2)
		date4 := now.AddDate(0, 0, -1)

		// 表示4戦: 勝ち・引き分け・勝ち・負け → 2勝1敗1分。
		// 勝率は引き分けを分母から除外して 2/(2+1)=0.666...
		rawMatches := []*entity.RecentMatch{
			entity.NewRecentMatch(0, date0, 0, "deck-01", "A", true, false, 0, "", "", nil),  // 前情報: 勝ち
			entity.NewRecentMatch(0, date1, 0, "deck-01", "B", true, false, 0, "", "", nil),  // 表示1: 勝ち
			entity.NewRecentMatch(0, date2, 0, "deck-01", "C", false, true, 0, "", "", nil),  // 表示2: 引き分け
			entity.NewRecentMatch(0, date3, 0, "deck-01", "D", true, false, 0, "", "", nil),  // 表示3: 勝ち
			entity.NewRecentMatch(0, date4, 0, "deck-01", "E", false, false, 0, "", "", nil), // 表示4: 負け
		}

		mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, fetchCount, deckId, uint(0)).Return(rawMatches, nil)
		mockEnvironmentRepository.EXPECT().FindByTerm(context.Background(), date0, date4).Return(nil, nil)

		ret, err := usecase.GetRecentMatches(context.Background(), userId, count, deckId, 0)

		require.NoError(t, err)
		require.Equal(t, 4, ret.TotalMatches)
		require.Equal(t, 2, ret.Wins)
		require.InDelta(t, 2.0/3.0, ret.WinRate, 0.0001, "勝率は引き分けを分母から除外する(2勝1敗1分 → 2/3)")

		// 表示2(引き分け)のローリング勝率: ウィンドウ[表示1:勝ち, 表示2:分]
		// = 勝ち1 / (決着1) = 1.0（引き分けは分母に入らない）
		require.InDelta(t, 1.0, ret.Matches[1].RollingWinRate, 0.0001)
	})
}

// 直近N戦に付ける環境ラベルも、開催日と実際の対戦環境がズレる公式イベント
// (official_event_environments)では登録された環境を使う。同じ日の対戦でも、
// 例外登録があるイベントと無いイベントで環境が分かれる。
func TestUserStatRecentUsecase_GetRecentMatches_EnvironmentOverride(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserStatRecentInterface(mockCtrl)
	mockEnvironmentRepository := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	mockOfficialEventEnvironmentRepository := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)
	usecase := NewUserStatRecent(mockRepository, mockEnvironmentRepository, mockOfficialEventEnvironmentRepository)

	userId := "user-03"

	// チャンピオンズリーグ2027横浜(m6として登録)と、同じ日のジムバトル(例外なし)
	const (
		championsLeagueEventId = uint(1113193)
		gymEventId             = uint(1113300)
	)
	eventDate := time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local)

	m6 := entity.NewEnvironment(
		"m6", "ストームエメラルダ",
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local),
		time.Date(2026, 9, 15, 0, 0, 0, 0, time.Local),
	)
	m6a := entity.NewEnvironment(
		"m6a", "30th CELEBRATION",
		time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local),
		time.Date(2026, 11, 26, 0, 0, 0, 0, time.Local),
	)

	rawMatches := []*entity.RecentMatch{
		entity.NewRecentMatch(0, eventDate, championsLeagueEventId, "deck-01", "対戦相手デッキA", true, false, 0, "", "", nil),
		entity.NewRecentMatch(0, eventDate, gymEventId, "deck-01", "対戦相手デッキB", false, false, 0, "", "", nil),
	}

	mockRepository.EXPECT().FindRecentMatches(context.Background(), userId, 2, "", uint(0)).Return(rawMatches, nil)
	// 対戦日(9/20)を含む期間の環境は m6a のみ
	mockEnvironmentRepository.EXPECT().FindByTerm(context.Background(), eventDate, eventDate).Return([]*entity.Environment{m6a}, nil)
	mockOfficialEventEnvironmentRepository.EXPECT().FindAll(context.Background()).Return(
		map[uint]string{championsLeagueEventId: "m6"}, nil,
	)
	mockEnvironmentRepository.EXPECT().FindById(context.Background(), "m6").Return(m6, nil)

	ret, err := usecase.GetRecentMatches(context.Background(), userId, 2, "", 0)

	require.NoError(t, err)
	require.Len(t, ret.Matches, 2)
	// 例外登録があるCLは、開催日が m6a の期間内でも m6
	require.Equal(t, "m6", ret.Matches[0].EnvironmentId)
	require.Equal(t, "ストームエメラルダ", ret.Matches[0].EnvironmentTitle)
	// 同じ日でも例外登録が無いジムバトルは開催日どおり m6a
	require.Equal(t, "m6a", ret.Matches[1].EnvironmentId)
	require.Equal(t, "30th CELEBRATION", ret.Matches[1].EnvironmentTitle)
}
