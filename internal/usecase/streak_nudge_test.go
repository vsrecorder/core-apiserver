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

// fixedNow は判定を安定させるための固定「現在時刻」。2026-07-26(日)。
// mondayOf(fixedNow) = 2026-07-20(月) が「今週の月曜」となる。
var fixedNow = time.Date(2026, 7, 26, 12, 0, 0, 0, time.Local)

// weeksAgoMonday は今週の月曜から n 週前の月曜(00:00)を返す。last_recorded_week の生成用。
func weeksAgoMonday(n int) time.Time {
	return mondayOf(fixedNow).AddDate(0, 0, -7*n)
}

func withFixedNow(t *testing.T) {
	t.Helper()
	original := timeNow
	timeNow = func() time.Time { return fixedNow }
	t.Cleanup(func() { timeNow = original })
}

func TestStreakNudge_NudgeUser(t *testing.T) {
	t.Run("正常系_2週あいてフリーズ空きあり(今週が瀬戸際)なら通知を作成する", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		// 最後の記録が2週前・フリーズ未使用 → 今週書けばフリーズ1枠で継続、書かなければ来週リセット
		stored := entity.NewUserStreak("user-1", 3, 5, 0, 0, weeksAgoMonday(2), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)
		notificationRepo.EXPECT().FindByUserId(gomock.Any(), "user-1", streakNudgeDedupScanLimit).Return(nil, nil)

		var saved *entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		)

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.NotNil(t, saved)
		require.Equal(t, NotificationCategoryStreak, saved.Category)
		require.Equal(t, streakNudgeTitle, saved.Title)
		require.Equal(t, streakNudgeLinkUrl, saved.LinkUrl)
		require.Contains(t, saved.Body, "3週連続") // current_weeks=3 が本文に反映される
	})

	t.Run("正常系_1週あいてフリーズ満杯(今週が瀬戸際)なら通知を作成する", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		// 先週記録・フリーズ満杯 → 今週書けば継続、書かなければ来週はフリーズが無く途切れる
		stored := entity.NewUserStreak("user-1", 6, 6, StreakMaxFreezeCount, 0, weeksAgoMonday(1), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)
		notificationRepo.EXPECT().FindByUserId(gomock.Any(), "user-1", streakNudgeDedupScanLimit).Return(nil, nil)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.True(t, sent)
	})

	t.Run("対象外_1週あいてフリーズ空きあり(まだ猶予がある)なら送らない", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		// 先週記録・フリーズ未使用 → 今週サボっても来週フリーズで救えるので瀬戸際ではない
		stored := entity.NewUserStreak("user-1", 2, 2, 0, 0, weeksAgoMonday(1), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)
		// isLastChanceThisWeek で早期に対象外となるため、通知の照会も保存も呼ばれない

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_今週すでに記録済みなら送らない", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		stored := entity.NewUserStreak("user-1", 4, 4, 0, 0, weeksAgoMonday(0), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_既に途切れている(3週あき)なら送らない", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		// 3週前が最後・フリーズ未使用 → フリーズ猶予(2週)を超えて既に途切れている
		stored := entity.NewUserStreak("user-1", 5, 5, 0, 0, weeksAgoMonday(3), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_今週すでにnudge済みなら2通目を作らない", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		stored := entity.NewUserStreak("user-1", 3, 5, 0, 0, weeksAgoMonday(2), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		// 今週の月曜以降に作られた nudge(同一見出し)が既に存在する
		existing := entity.NewNotification(
			"n-1", mondayOf(fixedNow).AddDate(0, 0, 1), "user-1",
			NotificationCategoryStreak, streakNudgeTitle, "既存の通知", streakNudgeLinkUrl,
		)
		notificationRepo.EXPECT().FindByUserId(gomock.Any(), "user-1", streakNudgeDedupScanLimit).
			Return([]*entity.Notification{existing}, nil)
		// Save は呼ばれない

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("正常系_先週の達成通知は同一週の判定に含めない(2通目を抑止しない)", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		stored := entity.NewUserStreak("user-1", 3, 5, 0, 0, weeksAgoMonday(2), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		// 先週作られた nudge と、達成通知(別見出し) → どちらも今週のnudge抑止には効かない
		lastWeekNudge := entity.NewNotification(
			"n-old", weeksAgoMonday(1), "user-1",
			NotificationCategoryStreak, streakNudgeTitle, "先週の通知", streakNudgeLinkUrl,
		)
		achievement := entity.NewNotification(
			"n-ach", mondayOf(fixedNow).AddDate(0, 0, 1), "user-1",
			NotificationCategoryStreak, "ストリークを継続中です", "達成", "/users",
		)
		notificationRepo.EXPECT().FindByUserId(gomock.Any(), "user-1", streakNudgeDedupScanLimit).
			Return([]*entity.Notification{achievement, lastWeekNudge}, nil)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.True(t, sent)
	})

	t.Run("dry-run_対象でも通知は作成せずtrueを返す", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		stored := entity.NewUserStreak("user-1", 3, 5, 0, 0, weeksAgoMonday(2), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)
		notificationRepo.EXPECT().FindByUserId(gomock.Any(), "user-1", streakNudgeDedupScanLimit).Return(nil, nil)
		// Save は呼ばれない

		sent, err := u.NudgeUser(context.Background(), "user-1", true)

		require.NoError(t, err)
		require.True(t, sent)
	})

	t.Run("対象外_ストリーク行が無いユーザーは送らない", func(t *testing.T) {
		withFixedNow(t)
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewStreakNudge(userStreakRepo, notificationRepo)

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)

		sent, err := u.NudgeUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})
}
