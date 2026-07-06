package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func newDesignationTestUsecase(mockCtrl *gomock.Controller) (
	*Designation,
	*mock_repository.MockDesignationInterface,
	*mock_repository.MockDesignationStatsInterface,
	*mock_repository.MockChampionshipSeriesInterface,
	*mock_repository.MockUserPlayerInterface,
) {
	designationRepo := mock_repository.NewMockDesignationInterface(mockCtrl)
	designationStatsRepo := mock_repository.NewMockDesignationStatsInterface(mockCtrl)
	championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)
	userPlayerRepo := mock_repository.NewMockUserPlayerInterface(mockCtrl)

	u := &Designation{designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo}

	return u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo
}

// expectUserPlayerNotLinked は、指定ユーザーがプレイヤーズクラブ未連携であることを表す
// FindByUserId の期待値を設定する(ベテランの達成条件を検証しない既存テスト用)。
func expectUserPlayerNotLinked(repo *mock_repository.MockUserPlayerInterface, userId string) {
	repo.EXPECT().FindByUserId(gomock.Any(), userId).Return(nil, apperror.ErrRecordNotFound)
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

// fourTierDefinitions は threeTierDefinitions に、レギュラー(tier4, シティリーグ記録1件。
// 「前シーズンに引き続き」の継続条件つき=前シーズンも1件以上必要。ただし今シーズン単独で
// DesignationCityLeagueStandaloneThreshold(2)件以上あれば継続条件なしでも達成扱い)を加えた4ティア。
func fourTierDefinitions(now time.Time) []*entity.Designation {
	return append(
		threeTierDefinitions(now),
		entity.NewDesignation("designation-04", 4, "regular", "🎫", "レギュラー", "", DesignationCriteriaTypeOfficialCityLeagueRecord, 1, now, now),
	)
}

// fiveTierDefinitions は fourTierDefinitions に、ベテラン(tier5。プレイヤーズクラブ連携済みの
// プレイヤーIDで cityleague_results に選択中シーズンの結果が1件以上あることが条件)を加えた
// 5ティア。
func fiveTierDefinitions(now time.Time) []*entity.Designation {
	return append(
		fourTierDefinitions(now),
		entity.NewDesignation("designation-05", 5, "veteran", "💪", "ベテラン", "", DesignationCriteriaTypeOfficialCityLeaguePlacement, 1, now, now),
	)
}

// sixTierDefinitions は fiveTierDefinitions に、熟練者(tier6。プレイヤーズクラブ連携済みの
// プレイヤーIDで cityleague_results に選択中シーズンの rank5以下の結果が1件以上あることが
// 条件)を加えた6ティア。
func sixTierDefinitions(now time.Time) []*entity.Designation {
	return append(
		fiveTierDefinitions(now),
		entity.NewDesignation("designation-06", 6, "expert", "🎖️", "熟練者", "", DesignationCriteriaTypeOfficialCityLeagueFinalTournament, 1, now, now),
	)
}

func TestDesignation_GetByUserId(t *testing.T) {
	t.Run("今シーズンの集計値が条件を満たすと現在の称号として返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

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
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

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
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

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

	t.Run("シティリーグ記録数が条件を満たし前シーズンも継続しているとレギュラー(tier4)まで到達する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil) // 今シーズン
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil) // 前シーズンも継続

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.True(t, item04.Achieved)
		require.Equal(t, 1, item04.CurrentValue)
		require.Equal(t, 1, item04.PreviousValue)
	})

	t.Run("前シーズンの実績が無くても今シーズン単独でシティリーグ記録が2件以上あればレギュラー(tier4)に到達する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil) // 今シーズン単独で2件
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil) // 前シーズンは無記録

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.True(t, item04.Achieved)
		require.Equal(t, 2, item04.CurrentValue)
		require.Equal(t, 0, item04.PreviousValue)
	})

	t.Run("シティリーグ記録が無いとレギュラー(tier4)には到達しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-03", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.False(t, item04.Achieved)
		require.Equal(t, 0, item04.CurrentValue)
		require.Equal(t, 0, item04.PreviousValue)
	})

	t.Run("今シーズンの記録が1件のみで前シーズンに記録が無ければレギュラー(tier4)には到達しない(継続条件も単独条件も未達)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fourTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil) // 今シーズンは1件のみ
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil) // 前シーズンは無記録

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-03", view.Current.ID)

		item04 := findDesignationLadderItem(view.Ladder, "designation-04")
		require.NotNil(t, item04)
		require.False(t, item04.Achieved)
		// CurrentValue は今シーズンの実際の集計値であり、継続条件の成否とは独立して表示される
		require.Equal(t, 1, item04.CurrentValue)
		require.Equal(t, 0, item04.PreviousValue)
	})

	t.Run("プレイヤーズクラブ未連携だとレギュラーの条件を満たしていてもベテラン(tier5)には到達しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		// シティリーグの記録(records)が既にあるにもかかわらず未連携、という状況を表す
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item05 := findDesignationLadderItem(view.Ladder, "designation-05")
		require.NotNil(t, item05)
		require.False(t, item05.Achieved)
		require.Equal(t, 0, item05.CurrentValue)
		require.True(t, item05.CityLeagueRecordWithoutPlayerLink)
	})

	t.Run("プレイヤーズクラブ未連携かつシティリーグの記録も無ければベテラン(tier5)の連携済み案内のヒントは立たない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-03", view.Current.ID)

		item05 := findDesignationLadderItem(view.Ladder, "designation-05")
		require.NotNil(t, item05)
		require.False(t, item05.Achieved)
		require.False(t, item05.CityLeagueRecordWithoutPlayerLink)
	})

	t.Run("プレイヤーズクラブ連携済みでもcityleague_resultsに一致するレコードが無ければベテラン(tier5)には到達しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)

		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultWithoutMatchingRecordByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item05 := findDesignationLadderItem(view.Ladder, "designation-05")
		require.NotNil(t, item05)
		require.False(t, item05.Achieved)
		require.Equal(t, 0, item05.CurrentValue)
		require.False(t, item05.MissingOfficialEventRecord)
	})

	t.Run("プレイヤーズクラブ連携済みでcityleague_resultsに存在するがofficial_event_idが一致するrecordsが無ければベテラン(tier5)には到達せず、記録不足のヒントが立つ", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)

		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultWithoutMatchingRecordByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(true, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item05 := findDesignationLadderItem(view.Ladder, "designation-05")
		require.NotNil(t, item05)
		require.False(t, item05.Achieved)
		require.True(t, item05.MissingOfficialEventRecord)
	})

	t.Run("プレイヤーズクラブ連携済みでcityleague_resultsに一致するレコードがあればベテラン(tier5)まで到達する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)

		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(true, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-05", view.Current.ID)

		item05 := findDesignationLadderItem(view.Ladder, "designation-05")
		require.NotNil(t, item05)
		require.True(t, item05.Achieved)
		require.Equal(t, 1, item05.CurrentValue)
		require.False(t, item05.MissingOfficialEventRecord)
	})

	t.Run("プレイヤーズクラブ未連携だとベテランの条件を満たしていても熟練者(tier6)には到達しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)
		expectUserPlayerNotLinked(userPlayerRepo, "user-1")

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(sixTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-04", view.Current.ID)

		item06 := findDesignationLadderItem(view.Ladder, "designation-06")
		require.NotNil(t, item06)
		require.False(t, item06.Achieved)
		require.Equal(t, 0, item06.CurrentValue)
		require.True(t, item06.CityLeagueRecordWithoutPlayerLink)
	})

	t.Run("プレイヤーズクラブ連携済みでもcityleague_resultsにrank5以下の一致するレコードが無ければ熟練者(tier6)には到達しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)

		designationRepo.EXPECT().FindAll(gomock.Any()).Return(sixTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(true, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-05", view.Current.ID)

		item06 := findDesignationLadderItem(view.Ladder, "designation-06")
		require.NotNil(t, item06)
		require.False(t, item06.Achieved)
		require.Equal(t, 0, item06.CurrentValue)
		require.False(t, item06.MissingOfficialEventRecord)
	})

	t.Run("プレイヤーズクラブ連携済みでcityleague_resultsにplayer_idは一致するがofficial_event_idが一致するrecordsが無ければ熟練者(tier6)には到達せず、記録不足のヒントが立つ", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)

		designationRepo.EXPECT().FindAll(gomock.Any()).Return(sixTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(true, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(true, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-05", view.Current.ID)

		item06 := findDesignationLadderItem(view.Ladder, "designation-06")
		require.NotNil(t, item06)
		require.False(t, item06.Achieved)
		require.True(t, item06.MissingOfficialEventRecord)
	})

	t.Run("プレイヤーズクラブ連携済みでcityleague_resultsにrank5以下の一致するレコードがあれば熟練者(tier6)まで到達する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)

		designationRepo.EXPECT().FindAll(gomock.Any()).Return(sixTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(true, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(true, nil)

		view, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
		require.NotNil(t, view.Current)
		require.Equal(t, "designation-06", view.Current.ID)

		item06 := findDesignationLadderItem(view.Ladder, "designation-06")
		require.NotNil(t, item06)
		require.True(t, item06.Achieved)
		require.Equal(t, 1, item06.CurrentValue)
	})

	t.Run("熟練者の判定はプレイヤーIDだけでなく選択中シーズンの期間でも絞り込まれる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)

		// expectCurrentAndPreviousChampionshipSeries が返す「今シーズン」の期間
		// (championshipSeriesDateRange により to_date の翌日0時がexclusive上限になる)。
		seasonFromDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local)
		seasonToDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)

		designationRepo.EXPECT().FindAll(gomock.Any()).Return(sixTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", seasonFromDate, seasonToDate).Return(true, nil)
		// 選択中シーズンの期間(fromDate/toDate)がそのまま渡っていることを検証する
		// (season の期間を無視して全期間を対象にしてしまう不具合の再発防止)。
		designationStatsRepo.EXPECT().
			ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, seasonFromDate, seasonToDate).
			Return(true, nil)

		_, err := u.GetByUserId(t.Context(), "user-1", "")

		require.NoError(t, err)
	})
}

func TestDesignation_GetRankStats(t *testing.T) {
	t.Run("ティアごとの到達ユーザー数と、いずれかのティアに到達した合計ユーザー数を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(sixTierDefinitions(now), nil)

		// expectCurrentAndPreviousChampionshipSeries が返す「今シーズン」の期間
		// (championshipSeriesDateRange により to_date の翌日0時がexclusive上限になる)。
		seasonFromDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local)
		seasonToDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)

		// user-1: 記録5件・リーグ0件・シティリーグ0件 -> tier2(見習い)
		// user-2: 記録5件・リーグ1件・シティリーグ4件(前シーズンも4件で継続)・
		//         プレイヤーズクラブ連携済みでシティリーグの結果(rank5以下)あり -> tier6(熟練者)
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
		designationStatsRepo.EXPECT().ExistsCityLeagueResultGroupByUserId(gomock.Any(), seasonFromDate, seasonToDate).
			Return(map[string]int{"user-2": 1}, nil)
		// 選択中シーズンの期間(fromDate/toDate)がそのまま渡っていることを検証する
		// (season の期間を無視して全期間を対象にしてしまう不具合の再発防止)。
		designationStatsRepo.EXPECT().
			ExistsCityLeagueFinalTournamentResultGroupByUserId(gomock.Any(), DesignationCityLeagueFinalTournamentMaxRank, seasonFromDate, seasonToDate).
			Return(map[string]int{"user-2": 1}, nil)

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
		require.Equal(t, 0, tierCounts[4])
		require.Equal(t, 0, tierCounts[5])
		require.Equal(t, 1, tierCounts[6])
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
