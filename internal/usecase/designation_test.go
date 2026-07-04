package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func newDesignationTestUsecase(mockCtrl *gomock.Controller) (
	*Designation,
	*mock_repository.MockDesignationInterface,
	*mock_repository.MockDesignationStatsInterface,
	*mock_repository.MockChampionshipSeriesInterface,
) {
	designationRepo := mock_repository.NewMockDesignationInterface(mockCtrl)
	designationStatsRepo := mock_repository.NewMockDesignationStatsInterface(mockCtrl)
	championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	u := &Designation{designationRepo, designationStatsRepo, championshipSeriesRepo}

	return u, designationRepo, designationStatsRepo, championshipSeriesRepo
}

// currentAndPreviousChampionshipSeries は、テスト対象の呼び出しが「今シーズン」「前シーズン」の
// championship_series を1回ずつ FindByDate で解決する前提でモックを設定する
// (previousSeasonRange は現在のシーズンのfrom_dateの前日から前シーズンを検索するため)。
func expectCurrentAndPreviousChampionshipSeries(repo *mock_repository.MockChampionshipSeriesInterface) {
	current := entity.NewChampionshipSeries(
		"series_2026", "チャンピオンシップシリーズ2026",
		time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
		time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local),
	)
	previous := entity.NewChampionshipSeries(
		"series_2025", "チャンピオンシップシリーズ2025",
		time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local),
		time.Date(2025, 8, 31, 0, 0, 0, 0, time.Local),
	)

	repo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(current, nil).AnyTimes()
	repo.EXPECT().FindByDate(gomock.Any(), time.Date(2025, 8, 31, 0, 0, 0, 0, time.Local)).Return(previous, nil).AnyTimes()
}

// threeTierDefinitions は 駆け出し(tier1, 記録1件)・見習い(tier2, 記録5件)・
// 一人前(tier3, 見習いの条件+リーグ記録)という累積構造を再現した3ティア。
func threeTierDefinitions(now time.Time) []*entity.Designation {
	return []*entity.Designation{
		entity.NewDesignation("designation-01", 1, "beginner", "🌱", "駆け出し", "", DesignationCriteriaTypeRecord, 1, now, now),
		entity.NewDesignation("designation-02", 2, "novice", "🔰", "見習い", "", DesignationCriteriaTypeRecord, 5, now, now),
		entity.NewDesignation("designation-03", 3, "independent", "👍", "一人前", "", DesignationCriteriaTypeOfficialLeagueRecord, 1, now, now),
	}
}

// fourTierDefinitions は threeTierDefinitions に、常連(tier4, シティリーグ記録4件。
// 「前シーズンに引き続き」の継続条件つき=前シーズンも4件以上必要)を加えた4ティア。
func fourTierDefinitions(now time.Time) []*entity.Designation {
	return append(
		threeTierDefinitions(now),
		entity.NewDesignation("designation-04", 4, "regular", "🎫", "常連", "", DesignationCriteriaTypeOfficialCityLeagueRecord, 4, now, now),
	)
}

func TestDesignation_GetByUserId(t *testing.T) {
	t.Run("今シーズンの集計値が条件を満たすと現在の称号として返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil) // 今シーズン
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil) // 前シーズン(常連の継続条件判定用)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-02", view.Current.ID)

		// ロードマップは tier1・tier2 が達成扱い、tier3 は未達成のまま
		for _, item := range view.Ladder {
			switch item.Designation.ID {
			case "designation-01", "designation-02":
				require.True(t, item.Achieved, item.Designation.ID)
				require.Equal(t, 5, item.CurrentValue, item.Designation.ID)
			case "designation-03":
				require.False(t, item.Achieved, item.Designation.ID)
				require.Equal(t, 0, item.CurrentValue, item.Designation.ID)
			}
		}
	})

	t.Run("シーズンが変わり集計値が0に戻れば称号なしになる(永続化された過去の実績は見ない)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// 新シーズンでまだ何も記録していない
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil) // 今シーズン
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil) // 前シーズン

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.Nil(t, view.Current)
		for _, item := range view.Ladder {
			require.False(t, item.Achieved, item.Designation.ID)
		}
	})

	t.Run("未実装(準備中)のティアは絶対に達成されない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		definitions := []*entity.Designation{
			entity.NewDesignation("designation-04", 4, "regular", "🎫", "常連", "", "unimplemented", 0, now, now),
		}
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(999, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(999, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(999, nil).Times(2)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.Nil(t, view.Current)
	})

	t.Run("シティリーグ記録数が条件を満たし前シーズンも継続していると常連(tier4)まで到達する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(4, nil) // 今シーズン
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(4, nil) // 前シーズンも継続

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.True(t, item04.Achieved)
		require.Equal(t, 4, item04.CurrentValue)
		require.Equal(t, 4, item04.PreviousValue)
	})

	t.Run("シティリーグ記録数が不足していると常連(tier4)には到達しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		// トレーナーズリーグの記録はあるが、シティリーグ単独では3件しかない
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(3, nil).Times(2)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-03", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.False(t, item04.Achieved)
		require.Equal(t, 3, item04.CurrentValue)
		require.Equal(t, 3, item04.PreviousValue)
	})

	t.Run("今シーズンの記録数は十分でも前シーズンに記録が無ければ常連(tier4)には到達しない(継続条件)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(4, nil) // 今シーズンは十分
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil) // 前シーズンは無記録

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-03", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.False(t, item04.Achieved)
		// CurrentValue は今シーズンの実際の集計値であり、継続条件の成否とは独立して表示される
		require.Equal(t, 4, item04.CurrentValue)
		require.Equal(t, 0, item04.PreviousValue)
	})
}

func TestDesignation_GetRankStats(t *testing.T) {
	t.Run("ティアごとの到達ユーザー数と、いずれかのティアに到達した合計ユーザー数を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)

		// user-1: 記録5件・リーグ0件・シティリーグ0件 -> tier2(見習い)
		// user-2: 記録5件・リーグ1件・シティリーグ4件(前シーズンも4件で継続) -> tier4(常連)
		// user-3: リーグ記録のみ1件(記録0件) -> 一つ目の条件(記録1件)すら
		//         満たさないため tier0(称号なし、集計対象外)
		designationStatsRepo.EXPECT().CountRecordsGroupByUserId(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(map[string]int{"user-1": 5, "user-2": 5}, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsGroupByUserId(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(map[string]int{"user-2": 1, "user-3": 1}, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsGroupByUserId(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(map[string]int{"user-2": 4}, nil) // 今シーズン
		designationStatsRepo.EXPECT().CountCityLeagueRecordsGroupByUserId(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(map[string]int{"user-2": 4}, nil) // 前シーズン(継続条件)

		view, err := u.GetRankStats(t.Context(), "")

		require.NoError(t, err)
		require.Equal(t, 2, view.TotalUsers)

		tierCounts := make(map[int]int)
		for _, t := range view.Tiers {
			tierCounts[t.Tier] = t.UserCount
		}
		require.Equal(t, 0, tierCounts[1])
		require.Equal(t, 1, tierCounts[2])
		require.Equal(t, 0, tierCounts[3])
		require.Equal(t, 1, tierCounts[4])
	})
}

func findDesignationLadderItem(ladder []*DesignationLadderItem, id string) *DesignationLadderItem {
	for _, item := range ladder {
		if item.Designation.ID == id {
			return item
		}
	}
	return nil
}
