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
) {
	badgeDefinitionRepo := mock_repository.NewMockBadgeDefinitionInterface(mockCtrl)
	userBadgeRepo := mock_repository.NewMockUserBadgeInterface(mockCtrl)
	userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
	badgeStatsRepo := mock_repository.NewMockBadgeStatsInterface(mockCtrl)

	u := &BadgeEvaluation{
		badgeDefinitionRepo: badgeDefinitionRepo,
		userBadgeRepo:       userBadgeRepo,
		userStreakRepo:      userStreakRepo,
		badgeStatsRepo:      badgeStatsRepo,
	}

	return u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo
}

func TestBadgeEvaluation_UpdateStreak(t *testing.T) {
	t.Run("初回記録は1週目として作成される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

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

	t.Run("翌週の記録は連続数が1増える", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 2, 2, 0, lastWeek, time.Now())

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

	t.Run("同じ週内の記録は連続数に影響しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		week := mondayOf(time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 2, 2, 0, week, time.Now())

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(current, nil)
		// 同一週内なので Save は呼ばれない

		sameWeekDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
		streak, err := u.updateStreak(context.Background(), "user-1", sameWeekDate, sameWeekDate)

		require.NoError(t, err)
		require.Equal(t, 2, streak.CurrentWeeks)
	})

	t.Run("1週分の空白はフリーズ枠を消費して連続扱いになる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 4, 4, 0, lastWeek, time.Now())

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

	t.Run("フリーズ枠を使い切った状態で2週空くとリセットされる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 4, 4, 1, lastWeek, time.Now())

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

	t.Run("3週間以上空くとリセットされる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		lastWeek := mondayOf(time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local))
		current := entity.NewUserStreak("user-1", 10, 10, 0, lastWeek, time.Now())

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
}

func TestBadgeEvaluation_EvaluateOnRecordCreated(t *testing.T) {
	t.Run("閾値に到達したバッジのみ新規付与する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

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

		// record_count=1 なので "初記録" のみ付与され、"10戦"・"3週連続" は付与されない
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.Equal(t, "def-first-record", ub.BadgeDefinitionId)
				return nil
			},
		).Times(1)

		record := entity.NewRecord("record-1", time.Now(), 0, "", "", "", "user-1", "", "", time.Now(), false, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-record", awarded[0].BadgeDefinitionId)
	})

	t.Run("既に獲得済みのバッジは再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
		}

		lastWeek := mondayOf(time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			entity.NewUserStreak("user-1", 1, 1, 0, lastWeek, now), nil,
		)

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-first-record", "record-0", now),
			}, nil,
		)

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil)
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない

		record := entity.NewRecord("record-2", now, 0, "", "", "", "user-1", "", "", now, false, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})

	t.Run("backfill等で過去日のrecordを再生した場合、achieved_atはevent_dateになり実行時刻にならない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

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

		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.True(t, ub.AchievedAt.Equal(pastEventDate))
				return nil
			},
		).Times(1)

		// created_at は「今backfillを実行した時刻」相当だが、achieved_atは実際の対戦日(event_date)を採用すべき
		record := entity.NewRecord("record-1", now, 0, "", "", "", "user-1", "", "", pastEventDate, false, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
	})
}

func TestBadgeEvaluation_EvaluateOnMatchCreated(t *testing.T) {
	t.Run("勝敗によらず初対戦バッジが付与される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-match", "first_match", "onboarding", "初対戦", "", "", BadgeCriteriaTypeMatchCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// 敗北した対戦(victoryFlg=false)でも「初対戦」は付与される
		match := entity.NewMatch("match-1", now, "record-1", "", "", "user-1", "", false, false, false, false, false, false, false, false, "", "", nil, nil)

		awarded, err := u.EvaluateOnMatchCreated(context.Background(), "user-1", match)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-match", awarded[0].BadgeDefinitionId)
	})

	t.Run("既に獲得済みなら再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

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

		match := entity.NewMatch("match-2", now, "record-1", "", "", "user-1", "", false, false, false, false, false, false, true, false, "", "", nil, nil)

		awarded, err := u.EvaluateOnMatchCreated(context.Background(), "user-1", match)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})
}

func TestBadgeEvaluation_EvaluateOnDeckCreated(t *testing.T) {
	t.Run("初デッキバッジが付与される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

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

		deck := entity.NewDeck("deck-1", now, time.Time{}, "user-1", "リザードンex", false, nil, nil)

		awarded, err := u.EvaluateOnDeckCreated(context.Background(), "user-1", deck)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-deck", awarded[0].BadgeDefinitionId)
	})
}

func TestBadgeEvaluation_EvaluateOnUserCreated(t *testing.T) {
	t.Run("ユーザー登録バッジが付与される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

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

		awarded, err := u.EvaluateOnUserCreated(context.Background(), "user-1", now)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-signup", awarded[0].BadgeDefinitionId)
	})

	t.Run("既に獲得済みなら再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

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
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない

		awarded, err := u.EvaluateOnUserCreated(context.Background(), "user-1", now)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})
}

func TestBadgeEvaluation_EvaluateOnRecordDeleted(t *testing.T) {
	t.Run("残っている記録の日付からストリークを作り直す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

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

	t.Run("最後の記録を削除すると連続数0で保存される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, badgeStatsRepo := newBadgeEvaluationTestUsecase(mockCtrl)

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
	t.Run("記録が無ければ全てゼロ値", func(t *testing.T) {
		currentWeeks, longestWeeks, freezeUsedCount, lastRecordedWeek := ComputeStreakState(nil)
		require.Equal(t, 0, currentWeeks)
		require.Equal(t, 0, longestWeeks)
		require.Equal(t, 0, freezeUsedCount)
		require.True(t, lastRecordedWeek.IsZero())
	})

	t.Run("連続した週なら最長連続数もそのまま反映される", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		currentWeeks, longestWeeks, freezeUsedCount, lastRecordedWeek := ComputeStreakState(dates)
		require.Equal(t, 3, currentWeeks)
		require.Equal(t, 3, longestWeeks)
		require.Equal(t, 0, freezeUsedCount)
		require.Equal(t, mondayOf(dates[2]), lastRecordedWeek)
	})

	t.Run("途中で途切れても最長記録は過去の値を保持する", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 5, 4, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 18, 0, 0, 0, 0, time.Local),
			// フリーズ枠を超えて大きく空白 → リセット
			time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local),
		}
		currentWeeks, longestWeeks, _, _ := ComputeStreakState(dates)
		require.Equal(t, 1, currentWeeks)
		require.Equal(t, 3, longestWeeks)
	})
}
