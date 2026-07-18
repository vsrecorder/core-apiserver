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

func TestStreak_GetByUserId(t *testing.T) {
	t.Run("正常系_記録が無いユーザーは0件のストリークを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, streak.CurrentWeeks)
		require.Equal(t, 0, streak.LongestWeeks)
	})

	t.Run("正常系_直近の記録から1週間以内ならそのまま返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		stored := entity.NewUserStreak("user-1", 3, 5, 0, mondayOf(time.Now()), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 3, streak.CurrentWeeks)
		require.Equal(t, 5, streak.LongestWeeks)
	})

	t.Run("正常系_フリーズ猶予(2週間)ちょうどでフリーズ未使用ならまだ継続扱い", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		lastWeek := mondayOf(time.Now()).AddDate(0, 0, -14)
		stored := entity.NewUserStreak("user-1", 4, 4, 0, lastWeek, time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 4, streak.CurrentWeeks)
	})

	t.Run("正常系_記録の作成・削除以来、時間経過だけでフリーズ猶予を超えた場合は表示上0に戻す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		// 5ヶ月前に最後の記録があり、以来新規記録も削除も無いまま user_streaks が
		// 更新されていない状態を再現する(本番で実際に観測された事例)。
		lastRecordedWeek := time.Date(2026, 2, 9, 0, 0, 0, 0, time.Local)
		stored := entity.NewUserStreak("user-1", 1, 8, 0, lastRecordedWeek, time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, streak.CurrentWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
		// 過去の最長記録は失われず保持される
		require.Equal(t, 8, streak.LongestWeeks)
	})

	t.Run("正常系_フリーズ猶予(2週間)を超え、かつフリーズ使用済みなら1週間経過時点でも終了扱い", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		lastWeek := mondayOf(time.Now()).AddDate(0, 0, -21)
		stored := entity.NewUserStreak("user-1", 2, 2, 1, lastWeek, time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, streak.CurrentWeeks)
	})
}

func TestIsStreakExpired(t *testing.T) {
	t.Run("正常系_記録が一度も無い(ゼロ値)場合は期限切れ扱いにしない", func(t *testing.T) {
		require.False(t, isStreakExpired(time.Time{}, 0))
	})

	t.Run("正常系_先週記録していれば今週分の記録がまだでも期限切れではない", func(t *testing.T) {
		lastWeek := mondayOf(time.Now()).AddDate(0, 0, -7)
		require.False(t, isStreakExpired(lastWeek, 0))
	})

	t.Run("正常系_3週間以上前で猶予を超えると期限切れ", func(t *testing.T) {
		lastWeek := mondayOf(time.Now()).AddDate(0, 0, -21)
		require.True(t, isStreakExpired(lastWeek, 0))
	})
}
