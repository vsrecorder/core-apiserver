package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func newDesignationEvaluationTestUsecase(mockCtrl *gomock.Controller) (
	*DesignationEvaluation,
	*mock_repository.MockDesignationInterface,
	*mock_repository.MockDesignationStatsInterface,
	*mock_repository.MockChampionshipSeriesInterface,
	*mock_repository.MockNotificationInterface,
) {
	designationRepo := mock_repository.NewMockDesignationInterface(mockCtrl)
	designationStatsRepo := mock_repository.NewMockDesignationStatsInterface(mockCtrl)
	championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)
	notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)

	u := &DesignationEvaluation{
		designationRepo:        designationRepo,
		designationStatsRepo:   designationStatsRepo,
		championshipSeriesRepo: championshipSeriesRepo,
		notificationRepo:       notificationRepo,
	}

	return u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo
}

// expectRecordCriteriaCounts はthreeTierDefinitionsを対象に、record/officialLeagueRecord/
// officialCityLeagueRecord(今シーズン・前シーズン)の集計値モックを設定する。
func expectRecordCriteriaCounts(
	designationStatsRepo *mock_repository.MockDesignationStatsInterface,
	userId string,
	recordCount int,
	leagueCount int,
) {
	designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), userId, gomock.Any(), gomock.Any()).Return(recordCount, nil)
	designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), userId, gomock.Any(), gomock.Any()).Return(leagueCount, nil)
	designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), userId, gomock.Any(), gomock.Any()).Return(0, nil).Times(2)
}

func TestDesignationEvaluation_CurrentTier(t *testing.T) {
	t.Run("集計値から現在のtierを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 5, 0)

		tier, err := u.CurrentTier(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 2, tier) // 見習い(記録5件以上)
	})

	t.Run("称号未達成なら0を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 0, 0)

		tier, err := u.CurrentTier(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, tier)
	})

	t.Run("シーズンが見つからない場合はエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, _, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		_, err := u.CurrentTier(context.Background(), "user-1")

		require.Error(t, err)
	})
}

func TestDesignationEvaluation_TierAsOf(t *testing.T) {
	t.Run("asOfをtoDateの上限として使う", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)

		asOf := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)

		var capturedToDate time.Time
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ time.Time, toDate time.Time) (int, error) {
				capturedToDate = toDate
				return 1, nil
			})
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		tier, err := u.TierAsOf(context.Background(), "user-1", asOf)

		require.NoError(t, err)
		require.Equal(t, 1, tier) // 駆け出し(記録1件以上)
		require.Equal(t, asOf, capturedToDate)
	})

	t.Run("asOfがシーズン終了日より後ならシーズン終了日を上限として使う", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)

		// expectCurrentAndPreviousChampionshipSeriesのcurrentシーズン(to_date=2026-08-31)
		// をchampionshipSeriesDateRangeに通した結果(翌日0時のexclusive上限)。
		seasonToDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		asOf := time.Date(2027, 1, 1, 0, 0, 0, 0, time.Local)

		var capturedToDate time.Time
		designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ time.Time, toDate time.Time) (int, error) {
				capturedToDate = toDate
				return 5, nil
			})
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		tier, err := u.TierAsOf(context.Background(), "user-1", asOf)

		require.NoError(t, err)
		require.Equal(t, 2, tier) // 見習い(記録5件以上)
		require.Equal(t, seasonToDate, capturedToDate)
	})
}

func TestDesignationEvaluation_NotifyIfTierChanged(t *testing.T) {
	t.Run("初めて称号(tier1)に到達すると称号獲得とランクアップの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 1, 0) // tier1(駆け出し)のみ達成

		var categories []string
		var bodies []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				categories = append(categories, n.Category)
				bodies = append(bodies, n.Body)
				return nil
			},
		).Times(2)

		u.NotifyIfTierChanged(context.Background(), "user-1", 0, now)

		require.ElementsMatch(t, []string{NotificationCategoryDesignation, NotificationCategoryRank}, categories)
		for _, body := range bodies {
			require.Contains(t, body, "2026シーズン") // どのシーズンの実績かを明記する
		}
	})

	t.Run("同じランク区分内でtierが上がった場合は称号獲得のみ通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		achievedAt := now.AddDate(0, 0, -5) // 実際に記録を作成した(=称号を達成した)過去の日時
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier1(駆け出し)・tier2(見習い、記録5件)はどちらも「モンスターボール級」
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 5, 0)

		var saved *entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		).Times(1)

		u.NotifyIfTierChanged(context.Background(), "user-1", 1, achievedAt)

		require.NotNil(t, saved)
		require.Equal(t, NotificationCategoryDesignation, saved.Category)
		require.Contains(t, saved.Body, "見習い")
		require.True(t, saved.CreatedAt.Equal(achievedAt)) // 通知の作成日時は実際の達成日時を使う
	})

	t.Run("ランク区分をまたいでtierが上がった場合は称号獲得とランクアップの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier3(一人前)は「スーパーボール級」(tier1・2の「モンスターボール級」とは異なる区分)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 5, 1)

		var bodies []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				bodies = append(bodies, n.Body)
				return nil
			},
		).Times(2)

		u.NotifyIfTierChanged(context.Background(), "user-1", 2, now)

		require.Len(t, bodies, 2)
	})

	t.Run("tierが変化していなければ何も通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 5, 0) // tier2のまま
		// notificationRepo.Save は呼ばれない(EXPECT未設定=呼ばれたら失敗)

		u.NotifyIfTierChanged(context.Background(), "user-1", 2, now)
	})

	t.Run("1回の評価で複数tierを一気に飛び越えた場合は通過した各tierをすべて通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier1(駆け出し)を飛ばしてtier0からtier3(一人前)まで一気に到達したケース
		// (称号機能の導入前から既に記録が5件・リーグ記録が1件あった場合等)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 5, 1)

		var categories []string
		var bodies []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				categories = append(categories, n.Category)
				bodies = append(bodies, n.Body)
				return nil
			},
		).Times(5) // 称号3件(駆け出し・見習い・一人前) + ランクアップ2件(モンスターボール級・スーパーボール級)

		u.NotifyIfTierChanged(context.Background(), "user-1", 0, now)

		require.Len(t, bodies, 5)
		designationCount := 0
		rankCount := 0
		for _, c := range categories {
			if c == NotificationCategoryDesignation {
				designationCount++
			}
			if c == NotificationCategoryRank {
				rankCount++
			}
		}
		require.Equal(t, 3, designationCount)
		require.Equal(t, 2, rankCount)

		foundBeginner := false
		for _, body := range bodies {
			if strings.Contains(body, "🌱 駆け出し") {
				foundBeginner = true
			}
		}
		require.True(t, foundBeginner, "途中で通過したtier1(駆け出し)の通知も欠落してはいけない")
	})

	t.Run("tierが下がった場合(シーズン切り替え等)は何も通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 5, 0) // tier2
		// notificationRepo.Save は呼ばれない

		u.NotifyIfTierChanged(context.Background(), "user-1", 3, now)
	})

	t.Run("シーズンが見つからない場合は何もせず終了する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, _, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)
		// notificationRepo.Save は呼ばれない

		u.NotifyIfTierChanged(context.Background(), "user-1", 0, now)
	})
}

func TestDesignationEvaluation_NotifyIfTierLost(t *testing.T) {
	t.Run("同じランク区分内でtierが下がった場合は称号喪失のみ通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// 記録削除により5件→1件に減り、tier2(見習い)からtier1(駆け出し)に後退
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 1, 0)

		var saved *entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		).Times(1)

		u.NotifyIfTierLost(context.Background(), "user-1", 2)

		require.NotNil(t, saved)
		require.Equal(t, NotificationCategoryDesignation, saved.Category)
		require.Contains(t, saved.Body, "見習い")
	})

	t.Run("ランク区分をまたいでtierが下がった場合は称号喪失とランクダウンの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier3(一人前、スーパーボール級)からtier1(駆け出し、モンスターボール級)に後退
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 1, 0)

		var bodies []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				bodies = append(bodies, n.Body)
				return nil
			},
		).Times(2)

		u.NotifyIfTierLost(context.Background(), "user-1", 3)

		require.Len(t, bodies, 2)
	})

	t.Run("称号が何もない状態まで失うと称号喪失とランクダウンの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 0, 0)

		var bodies []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				bodies = append(bodies, n.Body)
				return nil
			},
		).Times(2)

		u.NotifyIfTierLost(context.Background(), "user-1", 1)

		require.Len(t, bodies, 2)
	})

	t.Run("tierが変化していなければ何も通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, "user-1", 5, 0) // tier2のまま
		// notificationRepo.Save は呼ばれない

		u.NotifyIfTierLost(context.Background(), "user-1", 2)
	})

	t.Run("削除前のtierが0(称号未達成)なら何もしない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, _, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		// beforeTier<=0で即returnするため、designationRepo等は一切呼ばれない

		u.NotifyIfTierLost(context.Background(), "user-1", 0)
	})

	t.Run("シーズンが見つからない場合は何もせず終了する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, _, championshipSeriesRepo, _ := newDesignationEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)
		// notificationRepo.Save は呼ばれない

		u.NotifyIfTierLost(context.Background(), "user-1", 2)
	})
}

func TestRankNameForTier(t *testing.T) {
	require.Equal(t, "", RankNameForTier(0))
	require.Equal(t, "モンスターボール級", RankNameForTier(1))
	require.Equal(t, "モンスターボール級", RankNameForTier(2))
	require.Equal(t, "スーパーボール級", RankNameForTier(3))
	require.Equal(t, "スーパーボール級", RankNameForTier(4))
	require.Equal(t, "ハイパーボール級", RankNameForTier(5))
	require.Equal(t, "マスターボール級", RankNameForTier(6))
	require.Equal(t, "マスターボール級", RankNameForTier(8))
	require.Equal(t, "ウルトラボール級", RankNameForTier(9))
	require.Equal(t, "ウルトラボール級", RankNameForTier(10))
	require.Equal(t, "", RankNameForTier(11))
}
