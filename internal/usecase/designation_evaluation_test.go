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
	*mock_repository.MockUserPlayerInterface,
) {
	designationRepo := mock_repository.NewMockDesignationInterface(mockCtrl)
	designationStatsRepo := mock_repository.NewMockDesignationStatsInterface(mockCtrl)
	championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)
	notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
	userPlayerRepo := mock_repository.NewMockUserPlayerInterface(mockCtrl)

	u := &DesignationEvaluation{
		designationRepo:        designationRepo,
		designationStatsRepo:   designationStatsRepo,
		championshipSeriesRepo: championshipSeriesRepo,
		notificationRepo:       notificationRepo,
		userPlayerRepo:         userPlayerRepo,
	}

	return u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo
}

// expectRecordCriteriaCounts はthreeTierDefinitionsを対象に、record/officialLeagueRecord/
// officialCityLeagueRecord(今シーズン・前シーズン)の集計値モックを設定する。
func expectRecordCriteriaCounts(
	designationStatsRepo *mock_repository.MockDesignationStatsInterface,
	userPlayerRepo *mock_repository.MockUserPlayerInterface,
	userId string,
	recordCount int,
	leagueCount int,
) {
	designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), userId, gomock.Any(), gomock.Any()).Return(recordCount, nil)
	designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), userId, gomock.Any(), gomock.Any()).Return(leagueCount, nil)
	designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), userId, gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

	// 通知経路(CurrentTier/NotifyIfTierChanged/NotifyIfTierLost)もベテラン・熟練
	// (cityleague_results由来)のcriteriaを評価するようになったため、プレイヤーIDの連携有無を
	// 必ず引く。これらのテストは称号tier1〜3(threeTierDefinitions)の挙動を見るものなので、
	// 未連携(=ベテラン以降は未達成)として扱う。
	userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), userId).Return(nil, apperror.ErrRecordNotFound).AnyTimes()
}

func TestDesignationEvaluation_CurrentTier(t *testing.T) {
	t.Run("正常系_集計値から現在のtierを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 5, 0)

		tier, err := u.CurrentTier(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 2, tier) // 見習い(記録5件以上)
	})

	t.Run("正常系_称号未達成なら0を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 0, 0)

		tier, err := u.CurrentTier(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, tier)
	})

	t.Run("異常系_シーズンが見つからない場合はエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, _, championshipSeriesRepo, _, _ := newDesignationEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		_, err := u.CurrentTier(context.Background(), "user-1")

		require.Error(t, err)
	})
}

func TestDesignationEvaluation_TierAsOf(t *testing.T) {
	t.Run("正常系_asOfをtoDateの上限として使う", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)

		asOf := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)

		var capturedToDate time.Time
		designationStatsRepo.EXPECT().CountRecordsAsOfByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ time.Time, asOf time.Time) (int, error) {
				capturedToDate = asOf
				return 1, nil
			})
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		tier, err := u.TierAsOf(context.Background(), "user-1", asOf)

		require.NoError(t, err)
		require.Equal(t, 1, tier) // 駆け出し(記録1件以上)
		require.Equal(t, asOf, capturedToDate)
	})

	t.Run("正常系_asOfがシーズン終了日より後ならシーズン終了日を上限として使う", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)

		// expectCurrentAndPreviousChampionshipSeriesのcurrentシーズン(to_date=2026-08-31)
		// をchampionshipSeriesDateRangeに通した結果(翌日0時のexclusive上限)。
		seasonToDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		asOf := time.Date(2027, 1, 1, 0, 0, 0, 0, time.Local)

		var capturedToDate time.Time
		designationStatsRepo.EXPECT().CountRecordsAsOfByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ time.Time, asOf time.Time) (int, error) {
				capturedToDate = asOf
				return 5, nil
			})
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil)
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(0, nil).Times(2)

		tier, err := u.TierAsOf(context.Background(), "user-1", asOf)

		require.NoError(t, err)
		require.Equal(t, 2, tier) // 見習い(記録5件以上)
		require.Equal(t, seasonToDate, capturedToDate)
	})

	t.Run("正常系_cityleague_results起因のティア(ベテラン)にも到達できる", func(t *testing.T) {
		// backfill-notificationsがベテラン・ハイパーボール級の実際の達成日を
		// 遡って特定できるようにする回帰テスト。cityleague_results起因のcriteria_typeも
		// 含めて判定しなければならない(含めないとcurrentDesignation()がtier5未満で判定を
		// 打ち切ってしまい、ベテラン・ハイパーボール級のachieved_atが常にfallback=
		// バッチ実行時刻になる)。
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		designationStatsRepo.EXPECT().CountRecordsAsOfByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
		designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		// 今シーズン単独でDesignationCityLeagueStandaloneThreshold(2)件以上にして
		// レギュラー(tier4)を継続条件抜きで満たす。
		designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil).Times(2)

		userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
		userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueResultAsOfByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(true, nil)
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultAsOfByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)
		// 達人(優勝=rank1)。熟練と同じAsOfメソッドをしきい値1で流用するため、maxRank=1の呼び出しも期待する。
		designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultAsOfByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueChampionMaxRank, gomock.Any(), gomock.Any()).Return(false, nil)
		// 名人(常に入賞以上)。連携済みなら「入賞を逃したシティリーグ記録があるか」のAsOf版も必ず引く(このテストでは入賞漏れ記録なし=false)。
		designationStatsRepo.EXPECT().ExistsCityLeagueRecordWithoutPlacementAsOfByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(false, nil)

		asOf := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)
		tier, err := u.TierAsOf(context.Background(), "user-1", asOf)

		require.NoError(t, err)
		require.Equal(t, 5, tier) // ベテラン
		require.Equal(t, "ハイパーボール級", RankNameForTier(tier))
	})
}

// expectVeteranCriteria は「ベテラン(tier5)に到達しているユーザー」のリアルタイム評価用モックを
// 設定する。cityLeagueResultExists=false にすると、シティリーグ入賞の実績だけを失い
// レギュラー(tier4)へ後退した状態になる。
// リアルタイム評価(asOfゼロ)は表示側と同じ非AsOf版クエリを使うため、ここでも非AsOf版を期待する。
func expectVeteranCriteria(
	designationStatsRepo *mock_repository.MockDesignationStatsInterface,
	userPlayerRepo *mock_repository.MockUserPlayerInterface,
	now time.Time,
	cityLeagueResultExists bool,
) {
	designationStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(5, nil)
	designationStatsRepo.EXPECT().CountLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
	// 今シーズン単独でDesignationCityLeagueStandaloneThreshold(2)件以上にしてレギュラー(tier4)を満たす
	designationStatsRepo.EXPECT().CountCityLeagueRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil).Times(2)

	userPlayer := entity.NewUserPlayer("user-player-1", now, "user-1", "player-1")
	userPlayerRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(userPlayer, nil)
	designationStatsRepo.EXPECT().ExistsCityLeagueResultByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(cityLeagueResultExists, nil)
	designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueFinalTournamentMaxRank, gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	// 達人(優勝=rank1)。熟練と同じメソッドをしきい値1で流用するため、maxRank=1の呼び出しも期待する。
	designationStatsRepo.EXPECT().ExistsCityLeagueFinalTournamentResultByPlayerId(gomock.Any(), "user-1", "player-1", DesignationCityLeagueChampionMaxRank, gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	// 名人(常に入賞以上)。連携済みなら「入賞を逃したシティリーグ記録があるか」を必ず引く(このテストでは入賞漏れ記録なし=false)。
	designationStatsRepo.EXPECT().ExistsCityLeagueRecordWithoutPlacementByPlayerId(gomock.Any(), "user-1", "player-1", gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
}

// 通知経路(CurrentTier/NotifyIfTierChanged/NotifyIfTierLost)がcityleague_results起因の
// criteria_typeを評価しなかった頃は、currentDesignation()がtier5で判定を打ち切るため
// beforeTierが最大4に丸められ、ベテラン・熟練の獲得/剥奪/ランク変動の通知が一度も飛ばなかった。
// 表示側(usecase/designation.go)と判定を揃えたことの回帰テスト。
func TestDesignationEvaluation_CityLeagueResultTiersAreEvaluatedOnNotifyPath(t *testing.T) {
	t.Run("正常系_CurrentTierがベテラン(tier5)を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		expectVeteranCriteria(designationStatsRepo, userPlayerRepo, now, true)

		tier, err := u.CurrentTier(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 5, tier) // 修正前は4(レギュラー)に丸められていた
	})

	t.Run("正常系_ベテラン(tier5)からレギュラー(tier4)へ後退すると称号喪失を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(fiveTierDefinitions(now), nil)
		// シティリーグ入賞の実績を失い、tier5(ベテラン) -> tier4(レギュラー)へ後退
		expectVeteranCriteria(designationStatsRepo, userPlayerRepo, now, false)

		var saved []*entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				saved = append(saved, n)
				return nil
			},
		).AnyTimes()

		u.NotifyIfTierLost(context.Background(), "user-1", 5)

		// 修正前は beforeTier/afterTier とも4に丸められ、通知が1件も作られなかった
		require.NotEmpty(t, saved)
		require.Equal(t, NotificationCategoryDesignation, saved[0].Category)
		require.Contains(t, saved[0].Body, "ベテラン")
	})
}

func TestDesignationEvaluation_NotifyIfTierChanged(t *testing.T) {
	t.Run("正常系_初めて称号(tier1)に到達すると称号獲得とランクアップの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 1, 0) // tier1(駆け出し)のみ達成

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

	t.Run("正常系_同じランク区分内でtierが上がった場合は称号獲得のみ通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		achievedAt := now.AddDate(0, 0, -5) // 実際に記録を作成した(=称号を達成した)過去の日時
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier1(駆け出し)・tier2(見習い、記録5件)はどちらも「モンスターボール級」
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 5, 0)

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

	t.Run("正常系_ランク区分をまたいでtierが上がった場合は称号獲得とランクアップの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier3(一人前)は「スーパーボール級」(tier1・2の「モンスターボール級」とは異なる区分)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 5, 1)

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

	t.Run("正常系_tierが変化していなければ何も通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 5, 0) // tier2のまま
		// notificationRepo.Save は呼ばれない(EXPECT未設定=呼ばれたら失敗)

		u.NotifyIfTierChanged(context.Background(), "user-1", 2, now)
	})

	t.Run("正常系_1回の評価で複数tierを一気に飛び越えた場合は通過した各tierをすべて通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier1(駆け出し)を飛ばしてtier0からtier3(一人前)まで一気に到達したケース
		// (称号機能の導入前から既に記録が5件・リーグ記録が1件あった場合等)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 5, 1)

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

	t.Run("正常系_tierが下がった場合(シーズン切り替え等)は何も通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 5, 0) // tier2
		// notificationRepo.Save は呼ばれない

		u.NotifyIfTierChanged(context.Background(), "user-1", 3, now)
	})

	t.Run("異常系_シーズンが見つからない場合は何もせず終了する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, _, championshipSeriesRepo, _, _ := newDesignationEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)
		// notificationRepo.Save は呼ばれない

		u.NotifyIfTierChanged(context.Background(), "user-1", 0, now)
	})
}

func TestDesignationEvaluation_NotifyIfTierLost(t *testing.T) {
	t.Run("正常系_同じランク区分内でtierが下がった場合は称号喪失のみ通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// 記録削除により5件→1件に減り、tier2(見習い)からtier1(駆け出し)に後退
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 1, 0)

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

	t.Run("正常系_ランク区分をまたいでtierが下がった場合は称号喪失とランクダウンの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		// tier3(一人前、スーパーボール級)からtier1(駆け出し、モンスターボール級)に後退
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 1, 0)

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

	t.Run("正常系_称号が何もない状態まで失うと称号喪失とランクダウンの両方を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, notificationRepo, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 0, 0)

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

	t.Run("正常系_tierが変化していなければ何も通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, designationStatsRepo, championshipSeriesRepo, _, userPlayerRepo := newDesignationEvaluationTestUsecase(mockCtrl)
		expectCurrentAndPreviousChampionshipSeries(championshipSeriesRepo)

		now := time.Now()
		designationRepo.EXPECT().FindAll(gomock.Any()).Return(threeTierDefinitions(now), nil)
		expectRecordCriteriaCounts(designationStatsRepo, userPlayerRepo, "user-1", 5, 0) // tier2のまま
		// notificationRepo.Save は呼ばれない

		u.NotifyIfTierLost(context.Background(), "user-1", 2)
	})

	t.Run("正常系_削除前のtierが0(称号未達成)なら何もしない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, _, _, _ := newDesignationEvaluationTestUsecase(mockCtrl)
		// beforeTier<=0で即returnするため、designationRepo等は一切呼ばれない

		u.NotifyIfTierLost(context.Background(), "user-1", 0)
	})

	t.Run("異常系_シーズンが見つからない場合は何もせず終了する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, designationRepo, _, championshipSeriesRepo, _, _ := newDesignationEvaluationTestUsecase(mockCtrl)

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
	require.Equal(t, "ハイパーボール級", RankNameForTier(6)) // 熟練をマスターボール級からハイパーボール級へ移動
	require.Equal(t, "マスターボール級", RankNameForTier(7))
	require.Equal(t, "マスターボール級", RankNameForTier(8))
	require.Equal(t, "ウルトラボール級", RankNameForTier(9))
	require.Equal(t, "ウルトラボール級", RankNameForTier(10))
	require.Equal(t, "", RankNameForTier(11))
}
