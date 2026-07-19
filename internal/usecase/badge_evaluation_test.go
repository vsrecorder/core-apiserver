package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestMondayOf(t *testing.T) {
	// 2026-07-03 は金曜日 → 週の月曜日は 2026-06-29
	fri := time.Date(2026, 7, 3, 15, 30, 0, 0, time.Local)
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(fri))

	// 月曜日自身は同じ日を返す
	mon := time.Date(2026, 6, 29, 9, 0, 0, 0, time.Local)
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(mon))

	// 日曜日は前週の月曜日
	sun := time.Date(2026, 7, 5, 9, 0, 0, 0, time.Local)
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(sun))
}

func newBadgeEvaluationTestUsecase(mockCtrl *gomock.Controller) (
	*BadgeEvaluation,
	*mock_repository.MockBadgeDefinitionInterface,
	*mock_repository.MockUserBadgeInterface,
	*mock_repository.MockUserStreakInterface,
	*mock_repository.MockBadgeStatsInterface,
	*mock_repository.MockNotificationInterface,
	*mock_repository.MockChampionshipSeriesInterface,
) {
	badgeDefinitionRepo := mock_repository.NewMockBadgeDefinitionInterface(mockCtrl)
	userBadgeRepo := mock_repository.NewMockUserBadgeInterface(mockCtrl)
	userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
	badgeStatsRepo := mock_repository.NewMockBadgeStatsInterface(mockCtrl)
	notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
	championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	u := &BadgeEvaluation{
		badgeDefinitionRepo:    badgeDefinitionRepo,
		userBadgeRepo:          userBadgeRepo,
		userStreakRepo:         userStreakRepo,
		badgeStatsRepo:         badgeStatsRepo,
		notificationRepo:       notificationRepo,
		championshipSeriesRepo: championshipSeriesRepo,
	}

	return u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo
}

func TestBadgeEvaluation_UpdateStreak(t *testing.T) {
	t.Run("正常系_初回記録は1週目として作成される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 1, streak.CurrentWeeks)
				require.Equal(t, 1, streak.LongestWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				return nil
			},
		)

		eventDate := time.Date(2026, 7, 3, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", eventDate, eventDate)

		require.NoError(t, err)
		require.Equal(t, 1, streak.CurrentWeeks)
	})

	t.Run("正常系_翌週の記録は連続数が1増える", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 2, 2, 0, 0, lastWeek, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 3, streak.CurrentWeeks)
				require.Equal(t, 3, streak.LongestWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				return nil
			},
		)

		nextWeekDate := time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", nextWeekDate, nextWeekDate)

		require.NoError(t, err)
		require.Equal(t, 3, streak.CurrentWeeks)
	})

	t.Run("正常系_同じ週内の記録は連続数に影響しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		week := mondayOf(time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 2, 2, 0, 0, week, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		// 同一週内なので Save は呼ばれない

		sameWeekDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", sameWeekDate, sameWeekDate)

		require.NoError(t, err)
		require.Equal(t, 2, streak.CurrentWeeks)
	})

	t.Run("正常系_1週分の空白はフリーズ枠を消費して連続扱いになる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 4, 4, 0, 0, lastWeek, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 5, streak.CurrentWeeks)
				require.Equal(t, 1, streak.FreezeUsedCount)
				return nil
			},
		)

		// 2週間後(1週間の空白)
		twoWeeksLater := time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", twoWeeksLater, twoWeeksLater)

		require.NoError(t, err)
		require.Equal(t, 5, streak.CurrentWeeks)
	})

	t.Run("正常系_フリーズ枠を使い切った状態で2週空くとリセットされる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local))
		// フリーズ枠の上限は2。上限まで使い切った(FreezeUsedCount=2)状態を再現する。
		current := entity.NewUserStreak("user-1", 4, 4, 2, 0, lastWeek, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 1, streak.CurrentWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				require.Equal(t, 4, streak.LongestWeeks)
				return nil
			},
		)

		twoWeeksLater := time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", twoWeeksLater, twoWeeksLater)

		require.NoError(t, err)
		require.Equal(t, 1, streak.CurrentWeeks)
	})

	t.Run("正常系_3週間以上空くとリセットされる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 10, 10, 0, 0, lastWeek, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 1, streak.CurrentWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				return nil
			},
		)

		// フリーズ許容範囲(2週間)を超える4週間後
		muchLater := time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", muchLater, muchLater)

		require.NoError(t, err)
		require.Equal(t, 1, streak.CurrentWeeks)
	})

	t.Run("正常系_フリーズ枠は上限(2)まで消費できる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		// 既に1枠消費済み。もう1枠残っているので1週の空白でも継続扱いになる。
		lastWeek := mondayOf(time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 5, 5, 1, 0, lastWeek, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 6, streak.CurrentWeeks)
				require.Equal(t, 2, streak.FreezeUsedCount)
				return nil
			},
		)

		// 2週間後(1週間の空白) → 2枠目のフリーズを消費
		twoWeeksLater := time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", twoWeeksLater, twoWeeksLater)

		require.NoError(t, err)
		require.Equal(t, 6, streak.CurrentWeeks)
	})

	t.Run("正常系_フリーズを使わずstreakFreezeRegenWeeks週継続すると枠が1つ回復する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		// 1枠消費済み、回復まであと1週(進捗 = 回復間隔-1)。次のクリーンな週で1枠戻る。
		lastWeek := mondayOf(time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 8, 8, 1, streakFreezeRegenWeeks-1, lastWeek, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 9, streak.CurrentWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)     // 枠が回復
				require.Equal(t, 0, streak.FreezeRegenProgress) // 回復後は進捗リセット
				return nil
			},
		)

		// 翌週のクリーンな記録
		nextWeek := time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", nextWeek, nextWeek)

		require.NoError(t, err)
		require.Equal(t, 0, streak.FreezeUsedCount)
	})
}

func TestBadgeEvaluation_EvaluateOnRecordCreated(t *testing.T) {
	t.Run("正常系_閾値に到達したバッジのみ新規付与する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-record-10", "record_count_10", "milestone", "10戦", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "3週連続", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, now, now),
		}

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)

		// シーズンが見つからない場合はマイルストーン/週次ストリークの判定を黙ってスキップする
		// (record作成自体は失敗させない)ことをここで併せて確認する。
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		// record_count=1 なので "初記録" のみ付与され、"10戦"・"3週連続" は付与されない
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.Equal(t, "def-first-record", ub.BadgeDefinitionId)
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				require.Equal(t, "user-1", n.UserId)
				require.Equal(t, NotificationCategoryBadge, n.Category)
				return nil
			},
		).Times(1)

		record := entity.NewRecord("record-1", time.Now(), 0, "", "", "", "user-1", "", "", time.Now(), false, false, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-record", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_既に獲得済みのバッジは再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
		}

		lastWeek := mondayOf(time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			entity.NewUserStreak("user-1", 1, 1, 0, 0, lastWeek, now), nil,
		)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-first-record", "record-0", now),
			}, nil,
		)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil)
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない(=notificationRepo.Saveも呼ばれない)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		record := entity.NewRecord("record-2", now, 0, "", "", "", "user-1", "", "", now, false, false, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})

	t.Run("正常系_backfill等で過去日のrecordを再生した場合でも、achieved_atはcreated_at(記録した日時)になりevent_dateにならない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		pastEventDate := time.Date(2020, 1, 15, 0, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
		}

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.True(t, ub.AchievedAt.Equal(now))
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// event_dateは過去の対戦日(backfill入力値)だが、初記録バッジのachieved_atは
		// first_deck/first_match/signupと同様、実際に記録した日時(created_at)を採用すべき
		record := entity.NewRecord("record-1", now, 0, "", "", "", "user-1", "", "", pastEventDate, false, false, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
	})

	t.Run("正常系_マイルストーン系(record_count)は今回の記録でシーズン内の閾値をまたいだ場合のみ通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", "milestone", "10戦達成", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)

		// 1回目=オンボーディング判定用(全期間)、2回目=マイルストーン判定用(シーズンスコープ)。
		// milestone定義は無いためonboardingDefinitionsは空になり、award()の戻り値には影響しない。
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(10, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return([]time.Time{now}, nil)

		eventDate := now.AddDate(0, 0, -3) // 実際に対戦した(=達成した)日。記録の登録日時(now)とは別

		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				require.Equal(t, "user-1", n.UserId)
				require.Equal(t, NotificationCategoryBadge, n.Category)
				require.Contains(t, n.Body, "10戦達成")
				require.Contains(t, n.Body, "2026シーズン") // どのシーズンの実績かを明記する
				// 通知一覧の並び順が崩れないよう、通知のcreated_atは対戦日(event_date)ではなく
				// 実際の処理時刻(record.CreatedAt)を使う。
				require.True(t, n.CreatedAt.Equal(now))
				return nil
			},
		).Times(1)

		record := entity.NewRecord("record-10", now, 0, "", "", "", "user-1", "", "", eventDate, false, false, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Empty(t, awarded) // マイルストーン系はuser_badgesに永続化されない
	})

	t.Run("正常系_マイルストーン系(record_count)は閾値をまたいでいなければ通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", "milestone", "10戦達成", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)

		// まだ6件目(閾値10に届いていない) → notificationRepo.Saveは呼ばれない(EXPECT未設定=呼ばれたら失敗)
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(6, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return([]time.Time{now}, nil)

		record := entity.NewRecord("record-6", now, 0, "", "", "", "user-1", "", "", now, false, false, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})

	t.Run("正常系_週次ストリーク系はその週で最初の記録が閾値週数と一致する場合のみ通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		// 2026-06-15は月曜日(TestMondayOfで確認済みの2026-06-29から7日単位で遡って算出)
		thisWeekRecord := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "3週連続達成", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, thisWeekRecord, thisWeekRecord),
		}

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(3, nil).Times(2)

		// 過去2週+今回の記録(その週で最初)で3週連続
		seasonRecordDates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			thisWeekRecord,
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(seasonRecordDates, nil)

		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				require.Equal(t, NotificationCategoryStreak, n.Category)
				require.Contains(t, n.Body, "3週連続達成")
				return nil
			},
		).Times(1)

		record := entity.NewRecord("record-x", thisWeekRecord, 0, "", "", "", "user-1", "", "", thisWeekRecord, false, false, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
	})

	t.Run("正常系_onboarding系とマイルストーン系が同時に達成した場合、onboarding系の通知が先(=通知一覧では下)になる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		// 2026-06-15は月曜日(TestMondayOfで確認済みの2026-06-29から7日単位で遡って算出)
		thisWeekRecord := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, thisWeekRecord, thisWeekRecord),
			entity.NewBadgeDefinition("def-streak-1", "streak_week_1", "streak", "初週達成", "", "", BadgeCriteriaTypeStreakWeeks, 1, time.Time{}, time.Time{}, thisWeekRecord, thisWeekRecord),
		}

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)

		// 初めての記録: onboarding判定用(全期間)・マイルストーン判定用(シーズンスコープ)とも1件
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return([]time.Time{thisWeekRecord}, nil)

		var savedCategories []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				// onboarding系・マイルストーン系とも通知のcreated_atはrecord.CreatedAt
				// (=thisWeekRecord)で揃う。id DESCタイブレークで呼び出し順を制御するため。
				require.True(t, n.CreatedAt.Equal(thisWeekRecord))
				savedCategories = append(savedCategories, n.Category)
				return nil
			},
		).Times(2)

		record := entity.NewRecord("record-x", thisWeekRecord, 0, "", "", "", "user-1", "", "", thisWeekRecord, false, false, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
		require.Len(t, awarded, 1)

		// onboarding系(badge)が先に、マイルストーン系(streak)が後に保存される必要がある。
		// created_atが同値のため、id DESCタイブレークにより後に保存されたマイルストーン系の
		// 通知が通知一覧で上に、onboarding系の通知が一番下に表示される。
		require.Equal(t, []string{NotificationCategoryBadge, NotificationCategoryStreak}, savedCategories)
	})

	t.Run("正常系_同じ週の2件目の記録では週次ストリーク通知が重複しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		earlierThisWeek := time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local)
		secondRecordThisWeek := time.Date(2026, 6, 15, 18, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "3週連続達成", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, secondRecordThisWeek, secondRecordThisWeek),
		}

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			entity.NewUserStreak("user-1", 3, 3, 0, 0, mondayOf(earlierThisWeek), earlierThisWeek), nil,
		)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(4, nil).Times(2)

		// 同じ週に既に1件記録があるため、今回(2件目)は連続週数を進めていない
		seasonRecordDates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			earlierThisWeek,
			secondRecordThisWeek,
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(seasonRecordDates, nil)

		// notificationRepo.Saveは呼ばれない(EXPECT未設定=呼ばれたら失敗)

		record := entity.NewRecord("record-y", secondRecordThisWeek, 0, "", "", "", "user-1", "", "", secondRecordThisWeek, false, false, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
	})
}

func TestBadgeEvaluation_EvaluateOnMatchCreated(t *testing.T) {
	t.Run("正常系_勝敗によらず初対戦バッジが付与される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-match", "first_match", "onboarding", "初対戦", "", "", BadgeCriteriaTypeMatchCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		// 敗北した対戦(victoryFlg=false)でも「初対戦」は付与される
		match := entity.NewMatch("match-1", now, "record-1", "", "", "user-1", "", false, false, false, false, false, false, false, false, "", "", nil, nil)

		awarded, err := u.EvaluateOnMatchCreated(context.Background(), "user-1", match)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-match", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_既に獲得済みなら再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-match", "first_match", "onboarding", "初対戦", "", "", BadgeCriteriaTypeMatchCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-first-match", "", now),
			}, nil,
		)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil)
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		match := entity.NewMatch("match-2", now, "record-1", "", "", "user-1", "", false, false, false, false, false, false, true, false, "", "", nil, nil)

		awarded, err := u.EvaluateOnMatchCreated(context.Background(), "user-1", match)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})
}

func TestBadgeEvaluation_EvaluateOnDeckCreated(t *testing.T) {
	t.Run("正常系_初デッキバッジが付与される(デッキコード無し)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, notificationRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-deck", "first_deck", "onboarding", "初デッキ", "", "", BadgeCriteriaTypeDeckCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.Equal(t, "def-first-deck", ub.BadgeDefinitionId)
				require.Empty(t, ub.RecordId)
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// デッキコード無しで作成した場合、deck_codes は増えないため
		// マイルストーン系(deck_code_count)の判定は行われない
		// (=CountDeckCodesByUserId・championshipSeriesRepo は一切呼ばれない)。
		deck := entity.NewDeck("deck-1", now, time.Time{}, "user-1", "リザードンex", false, nil, nil)

		awarded, err := u.EvaluateOnDeckCreated(context.Background(), "user-1", deck)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-deck", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_デッキコード付きで作成した場合はマイルストーン系(deck_code_count)も判定する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-deck", "first_deck", "onboarding", "初デッキ", "", "", BadgeCriteriaTypeDeckCount, 1, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-deck-code-1", "deck_code_count_1", "milestone", "駆け出しビルダー", "", "", BadgeCriteriaTypeDeckCodeCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		badgeStatsRepo.EXPECT().CountDeckCodesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(2)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil).Times(2)

		deckCode := entity.NewDeckCode("deckcode-1", now, "user-1", "deck-1", "AbCdEfGhIj12", false, "")
		deck := entity.NewDeck("deck-1", now, time.Time{}, "user-1", "リザードンex", false, deckCode, nil)

		awarded, err := u.EvaluateOnDeckCreated(context.Background(), "user-1", deck)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-deck", awarded[0].BadgeDefinitionId)
	})
}

func TestBadgeEvaluation_EvaluateOnUserCreated(t *testing.T) {
	t.Run("正常系_ユーザー登録バッジが付与される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, _, notificationRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-signup", "signup", "onboarding", "バトレコユーザー", "", "", BadgeCriteriaTypeSignup, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.Equal(t, "def-signup", ub.BadgeDefinitionId)
				require.Empty(t, ub.RecordId)
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		awarded, err := u.EvaluateOnUserCreated(context.Background(), "user-1", now)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-signup", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_既に獲得済みなら再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-signup", "signup", "onboarding", "バトレコユーザー", "", "", BadgeCriteriaTypeSignup, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-signup", "", now),
			}, nil,
		)
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない(=notificationRepo.Saveも呼ばれない)

		awarded, err := u.EvaluateOnUserCreated(context.Background(), "user-1", now)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})
}

func TestBadgeEvaluation_EvaluateOnRecordDeleted(t *testing.T) {
	t.Run("正常系_残っている記録の日付からストリークを作り直す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, badgeStatsRepo, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		// 3週連続していたうち、直近1週分の記録を削除して2週連続に減った想定
		remaining := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(remaining, nil)

		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 2, streak.CurrentWeeks)
				require.Equal(t, 2, streak.LongestWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				require.Equal(t, mondayOf(remaining[1]), streak.LastRecordedWeek)
				return nil
			},
		)

		err := u.EvaluateOnRecordDeleted(context.Background(), "user-1")

		require.NoError(t, err)
	})

	t.Run("正常系_最後の記録を削除すると連続数0で保存される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, badgeStatsRepo, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(nil, nil)

		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 0, streak.CurrentWeeks)
				require.Equal(t, 0, streak.LongestWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				require.True(t, streak.LastRecordedWeek.IsZero())
				return nil
			},
		)

		err := u.EvaluateOnRecordDeleted(context.Background(), "user-1")

		require.NoError(t, err)
	})
}

func TestComputeStreakState(t *testing.T) {
	t.Run("正常系_記録が無ければ全てゼロ値", func(t *testing.T) {
		currentWeeks, longestWeeks, freezeUsedCount, _, lastRecordedWeek := ComputeStreakState(nil)
		require.Equal(t, 0, currentWeeks)
		require.Equal(t, 0, longestWeeks)
		require.Equal(t, 0, freezeUsedCount)
		require.True(t, lastRecordedWeek.IsZero())
	})

	t.Run("正常系_連続した週なら最長連続数もそのまま反映される", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		currentWeeks, longestWeeks, freezeUsedCount, _, lastRecordedWeek := ComputeStreakState(dates)
		require.Equal(t, 3, currentWeeks)
		require.Equal(t, 3, longestWeeks)
		require.Equal(t, 0, freezeUsedCount)
		require.Equal(t, mondayOf(dates[2]), lastRecordedWeek)
	})

	t.Run("正常系_途中で途切れても最長記録は過去の値を保持する", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 5, 4, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 18, 0, 0, 0, 0, time.Local),
			// フリーズ枠を超えて大きく空白 → リセット
			time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local),
		}
		currentWeeks, longestWeeks, _, _, _ := ComputeStreakState(dates)
		require.Equal(t, 1, currentWeeks)
		require.Equal(t, 3, longestWeeks)
	})

	t.Run("正常系_フリーズ消費後にstreakFreezeRegenWeeks週クリーン継続すると枠が回復する", func(t *testing.T) {
		// 6/8を飛ばして6/1→6/15でフリーズを1枠消費、その後クリーンな週を
		// streakFreezeRegenWeeks週続けると使用済み枠が0に戻る。
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			// 6/8 は未記録 → 6/15 でフリーズ消費
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		clean := time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local)
		for w := 0; w < streakFreezeRegenWeeks; w++ {
			dates = append(dates, clean.AddDate(0, 0, 7*w))
		}

		_, _, freezeUsedCount, freezeRegenProgress, _ := ComputeStreakState(dates)
		require.Equal(t, 0, freezeUsedCount)
		require.Equal(t, 0, freezeRegenProgress)
	})
}

func TestStreakWeeksAchievedAt(t *testing.T) {
	t.Run("正常系_記録が無ければnil", func(t *testing.T) {
		require.Nil(t, StreakWeeksAchievedAt(nil))
	})

	t.Run("正常系_連続した週は週数ごとに初めて到達した日を返す", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		achievedAt := StreakWeeksAchievedAt(dates)
		require.Equal(t, dates[0], achievedAt[1])
		require.Equal(t, dates[1], achievedAt[2])
		require.Equal(t, dates[2], achievedAt[3])
	})

	t.Run("正常系_途切れてリセットされても最初に到達した日はそのまま残る", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 5, 4, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 18, 0, 0, 0, 0, time.Local),
			// フリーズ枠を超えて大きく空白 → リセットして再び1週目から
			time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local),
		}
		achievedAt := StreakWeeksAchievedAt(dates)
		require.Equal(t, dates[0], achievedAt[1])
		require.Equal(t, dates[1], achievedAt[2])
		require.Equal(t, dates[2], achievedAt[3])
		require.NotContains(t, achievedAt, 4)
	})

	t.Run("正常系_同じ週内の複数記録はその週で最も早い日付を使う", func(t *testing.T) {
		monday := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		dates := []time.Time{
			monday.AddDate(0, 0, 3), // 同じ週の木曜
			monday,                  // 同じ週の月曜(最も早い)
		}
		achievedAt := StreakWeeksAchievedAt(dates)
		require.Equal(t, monday, achievedAt[1])
	})
}
