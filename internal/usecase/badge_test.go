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
	t.Run("オンボーディング系は永続化された獲得記録をそのまま参照する(シーズン集計値は使わない)", func(t *testing.T) {
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

	t.Run("マイルストーン系は今シーズンの集計値のみでライブ判定する(過去の獲得記録は見ない)", func(t *testing.T) {
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
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-record-10")
		require.NotNil(t, view)
		require.True(t, view.Achieved)
		require.Equal(t, 10, view.CurrentValue)
		require.True(t, view.AchievedAt.IsZero(), "日付一覧が閾値に満たない場合はachieved_atを求めない")
	})

	t.Run("マイルストーン系は今シーズン内でcriteria_value番目に到達した日付をachieved_atとして返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-3", "record_count_3", BadgeCategoryMilestone, "駆け出しユーザー", "", "", BadgeCriteriaTypeRecordCount, 3, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-deck-2", "deck_count_2", BadgeCategoryMilestone, "駆け出しビルダー", "", "", BadgeCriteriaTypeDeckCount, 2, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-match-2", "match_count_2", BadgeCategoryMilestone, "駆け出しバトラー", "", "", BadgeCriteriaTypeMatchCount, 2, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(3, nil).Times(2)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil).Times(2)

		recordDate3 := time.Date(2026, 6, 20, 0, 0, 0, 0, time.Local)
		deckDate2 := time.Date(2026, 5, 10, 0, 0, 0, 0, time.Local)
		matchDate2 := time.Date(2026, 6, 25, 0, 0, 0, 0, time.Local)

		// FindXxxDatesByUserId は昇順であることを前提にせず usecase 側でソートするため、
		// あえて逆順で返してソートの必要性を検証する。
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(
			[]time.Time{recordDate3, recordDate3.AddDate(0, 0, -20), recordDate3.AddDate(0, 0, -40)}, nil,
		)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(
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

	t.Run("週次ストリーク系は今シーズンの記録日から連続週数を計算して判定する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", BadgeCategoryStreak, "週次記録3週連続", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		// 3週連続分の記録日(月曜始まり基準で3週分)
		week1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		week2 := time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local)
		week3 := time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).
			Return([]time.Time{week1, week2, week3}, nil)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-streak-3")
		require.NotNil(t, view)
		require.True(t, view.Achieved)
		require.Equal(t, 3, view.CurrentValue)
		require.True(t, view.AchievedAt.Equal(week3), "3週連続に初めて到達した週の記録日がachieved_atになる")
	})

	t.Run("週次ストリーク系はストリークが途切れても、シーズン内で最初に到達した日付を保持する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, badgeStatsRepo, championshipSeriesRepo := newBadgeTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-2", "streak_week_2", BadgeCategoryStreak, "週次記録2週連続", "", "", BadgeCriteriaTypeStreakWeeks, 2, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		// 1,2週目で2週連続を達成した後、大きく空いて途切れ、直近は単発(1週)のみの状態。
		// 「現在」は2週連続ではない(Achieved=false)が、シーズン内で最初に2週連続に到達した
		// 日付(week2)は保持され続ける。
		week1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		week2 := time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local)
		recentWeek := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).
			Return([]time.Time{week1, week2, recentWeek}, nil)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		view := findView(views, "def-streak-2")
		require.NotNil(t, view)
		require.False(t, view.Achieved, "直近は単発記録のみで現在は2週連続ではない")
		require.True(t, view.AchievedAt.Equal(week2), "途切れていても、シーズン内で最初に2週連続に到達した日付は残る")
	})

	t.Run("season指定時はそのシーズンの期間で集計する", func(t *testing.T) {
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
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindDeckDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)
		badgeStatsRepo.EXPECT().FindMatchDatesByUserId(gomock.Any(), "user-1", wantFrom, wantTo).Return(nil, nil)

		views, err := u.GetByUserId(t.Context(), "user-1", "2024")

		require.NoError(t, err)
		view := findView(views, "def-record-10")
		require.NotNil(t, view)
		require.Equal(t, 3, view.CurrentValue)
	})
}

func TestSeasonStreakWeeks(t *testing.T) {
	t.Run("記録が無ければ0", func(t *testing.T) {
		require.Equal(t, 0, seasonStreakWeeks(nil))
	})

	t.Run("連続した週なら連続数がそのまま返る", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		require.Equal(t, 3, seasonStreakWeeks(dates))
	})

	t.Run("1週空いてもフリーズ枠で連続扱いになる", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local), // 2週間後(1週間の空白)
		}
		require.Equal(t, 2, seasonStreakWeeks(dates))
	})

	t.Run("フリーズ枠を超えて空くと直近の連続数だけが残る", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local), // 大きく空白
			time.Date(2026, 7, 27, 0, 0, 0, 0, time.Local),
		}
		require.Equal(t, 2, seasonStreakWeeks(dates))
	})
}
