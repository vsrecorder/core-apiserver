package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func newBadgeTestUsecase(mockCtrl *gomock.Controller) (
	*Badge,
	*mock_repository.MockBadgeDefinitionInterface,
	*mock_repository.MockUserBadgeInterface,
	*mock_repository.MockBadgeStatsInterface,
	*mock_repository.MockChampionshipSeriesInterface,
) {
	badgeDefinitionRepo := mock_repository.NewMockBadgeDefinitionInterface(mockCtrl)
	userBadgeRepo := mock_repository.NewMockUserBadgeInterface(mockCtrl)
	badgeStatsRepo := mock_repository.NewMockBadgeStatsInterface(mockCtrl)
	championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	u := &Badge{
		badgeDefinitionRepo:    badgeDefinitionRepo,
		userBadgeRepo:          userBadgeRepo,
		badgeStatsRepo:         badgeStatsRepo,
		championshipSeriesRepo: championshipSeriesRepo,
	}

	return u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo
}

// currentChampionshipSeries はテスト用の「現在のシーズン」を返す(具体的な期間の値は
// season空文字時のテストでは検証対象ではないため、固定の1シーズン分を使い回す)。
func currentChampionshipSeries() *entity.ChampionshipSeries {
	return entity.NewChampionshipSeries(
		"series_2026", "チャンピオンシップシリーズ2026",
		time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
		time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local),
	)
}

func findView(views []*UserBadgeView, id string) *UserBadgeView {
	for _, v := range views {
		if v.Definition.ID == id {
			return v
		}
	}
	return nil
}

func TestBadge_GetByUserId(t *testing.T) {
	t.Run("正常系_オンボーディング系は永続化された獲得記録をそのまま参照する(シーズン集計値は使わない)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", BadgeCategoryOnboarding, "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-first-record", "record-1", now),
			}, nil,
		)
		// オンボーディングは全期間、マイルストーン/ストリークは今シーズンの2種類の集計値を取得する
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDeckCodesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		badgeStatsRepo.EXPECT().FindDeckCodeDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-first-record")
		require.NotNil(t, view)
		require.True(t, view.Achieved)
		require.Equal(t, now.Unix(), view.AchievedAt.Unix())
	})

	t.Run("正常系_マイルストーン系は今シーズンの集計値のみでライブ判定する(過去の獲得記録は見ない)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", BadgeCategoryMilestone, "駆け出しユーザー", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil)
		// user_badges には何も無い(永続化していない)が、今シーズンの記録数が10件あるので達成扱いになる
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		// 全期間カウント(オンボーディング用、呼ばれるが今回は未使用)
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(999, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)

		// 今シーズンのカウント(マイルストーン用)。CountRecordsByUserId(=10)と
		// FindRecordDatesByUserId(日付一覧)は独立にモックしているため、日付一覧の方は
		// 閾値に満たない(=achieved_atは求まらない)。日付一覧が伴う場合の挙動は別テストで検証する。
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Not(time.Time{})).Return(10, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Not(time.Time{})).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Not(time.Time{})).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDeckCodesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Not(time.Time{})).Return(0, nil)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckCodeDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-record-10")
		require.NotNil(t, view)
		require.True(t, view.Achieved)
		require.Equal(t, 10, view.CurrentValue)
		require.True(t, view.AchievedAt.IsZero(), "日付一覧が閾値に満たない場合はachieved_atを求めない")
	})

	t.Run("正常系_マイルストーン系は今シーズン内でcriteria_value番目に到達した日付をachieved_atとして返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-3", "record_count_3", BadgeCategoryMilestone, "駆け出しユーザー", "", "", BadgeCriteriaTypeRecordCount, 3, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-deck-2", "deck_count_2", BadgeCategoryMilestone, "駆け出しビルダー", "", "", BadgeCriteriaTypeDeckCodeCount, 2, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-match-2", "match_count_2", BadgeCategoryMilestone, "駆け出しバトラー", "", "", BadgeCriteriaTypeMatchCount, 2, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(3, nil).Times(2)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDeckCodesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil)

		recordDate3 := time.Date(2026, 6, 20, 0, 0, 0, 0, time.Local)
		deckDate2 := time.Date(2026, 5, 10, 0, 0, 0, 0, time.Local)
		matchDate2 := time.Date(2026, 6, 25, 0, 0, 0, 0, time.Local)

		// FindXxxDatesByUserId は昇順であることを前提にせず usecase 側でソートするため、
		// あえて逆順で返してソートの必要性を検証する。
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(
			[]time.Time{recordDate3, recordDate3.AddDate(0, 0, -20), recordDate3.AddDate(0, 0, -40)}, nil,
		)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckCodeDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(
			[]time.Time{deckDate2, deckDate2.AddDate(0, 0, -5)}, nil,
		)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(
			[]time.Time{matchDate2, matchDate2.AddDate(0, 0, -3)}, nil,
		)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)

		recordView := findView(views, "def-record-3")
		require.NotNil(t, recordView)
		require.True(t, recordView.Achieved)
		require.True(t, recordView.AchievedAt.Equal(recordDate3), "3番目に古い記録日がachieved_atになる")

		deckView := findView(views, "def-deck-2")
		require.NotNil(t, deckView)
		require.True(t, deckView.Achieved)
		require.True(t, deckView.AchievedAt.Equal(deckDate2), "2番目に古いデッキ登録日がachieved_atになる")

		matchView := findView(views, "def-match-2")
		require.NotNil(t, matchView)
		require.True(t, matchView.Achieved)
		require.True(t, matchView.AchievedAt.Equal(matchDate2), "2番目に古い対戦日がachieved_atになる")
	})

	t.Run("正常系_season指定時はそのシーズンの期間で集計する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", BadgeCategoryMilestone, "駆け出しユーザー", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}
		wantFrom := time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local)
		wantTo := time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		championshipSeriesRepo.EXPECT().FindById(gomock.Any(), "series_2024").Return(
			entity.NewChampionshipSeries("series_2024", "チャンピオンシップシリーズ2024", wantFrom, time.Date(2024, 8, 31, 0, 0, 0, 0, time.Local)),
			nil,
		)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(0, nil)

		// 2024シーズン(2023-09-01〜2024-08-31)がそのまま渡されることを検証する
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(3, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(0, nil)
		badgeStatsRepo.EXPECT().CountDeckCodesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(0, nil)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckCodeDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "2024")

		require.NoError(t, err)
		view := findView(views, "def-record-10")
		require.NotNil(t, view)
		require.Equal(t, 3, view.CurrentValue)
	})
}
