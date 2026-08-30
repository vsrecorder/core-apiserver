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

		stored := entity.NewUserStreak("user-1", 3, 5, 0, 0, mondayOf(time.Now()), time.Now())
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
		stored := entity.NewUserStreak("user-1", 4, 4, 0, 0, lastWeek, time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 4, streak.CurrentWeeks)
		// 先週の空き週ぶんのフリーズは消費済みとして返す
		require.Equal(t, 1, streak.FreezeUsedCount)
	})

	t.Run("正常系_先週が未記録でフリーズに空きがあれば空き週ぶんのフリーズを消費済みとして返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		// 8/17 週まで6週連続、8/24 週は未記録、今日は 8/31(月)。保存値はまだ 8/17 時点のままだが、
		// 8/24 週の空きは次に記録した時点で必ずフリーズ1つを消費するので、表示では先に減らす。
		// フリーズ消費で回復進捗も 0 に戻る(ComputeStreakState と同じ)。
		overrideTimeNow(t, time.Date(2026, 8, 31, 10, 0, 0, 0, time.Local))
		stored := entity.NewUserStreak("user-1", 6, 6, 0, 1, time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 6, streak.CurrentWeeks)
		require.Equal(t, 6, streak.LongestWeeks)
		require.Equal(t, 1, streak.FreezeUsedCount)
		require.Equal(t, 0, streak.FreezeRegenProgress)
		require.Equal(t, stored.LastRecordedWeek, streak.LastRecordedWeek)
	})

	t.Run("正常系_先週記録していれば今週未記録でもフリーズは消費しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		// 8/24 週に記録済み、今日は 8/31(月)。今週はまだこれから記録できるので空き週ではない。
		overrideTimeNow(t, time.Date(2026, 8, 31, 10, 0, 0, 0, time.Local))
		stored := entity.NewUserStreak("user-1", 6, 6, 1, 1, time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 6, streak.CurrentWeeks)
		require.Equal(t, 1, streak.FreezeUsedCount)
		require.Equal(t, 1, streak.FreezeRegenProgress)
	})

	t.Run("正常系_先週が未記録でフリーズ満杯なら消費できず終了扱いにする", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		// 先読みの消費で上限を超えることはなく、空きが無ければ途切れ扱い(0週・フリーズ未使用)。
		overrideTimeNow(t, time.Date(2026, 8, 31, 10, 0, 0, 0, time.Local))
		stored := entity.NewUserStreak("user-1", 6, 8, StreakMaxFreezeCount, 0, time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, streak.CurrentWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
		require.Equal(t, 8, streak.LongestWeeks)
	})

	t.Run("正常系_記録の作成・削除以来、時間経過だけでフリーズ猶予を超えた場合は表示上0に戻す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		// 5ヶ月前に最後の記録があり、以来新規記録も削除も無いまま user_streaks が
		// 更新されていない状態を再現する(本番で実際に観測された事例)。
		lastRecordedWeek := time.Date(2026, 2, 9, 0, 0, 0, 0, time.Local)
		stored := entity.NewUserStreak("user-1", 1, 8, 0, 0, lastRecordedWeek, time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, streak.CurrentWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
		// 過去の最長記録は失われず保持される
		require.Equal(t, 8, streak.LongestWeeks)
	})

	t.Run("正常系_記録削除で最終記録週が3週前まで戻った場合はDB由来のUTC日付でも終了扱いにする", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		// 8/3 週まで5週連続していた人が 8/24 週に記録すると猶予超えで1週目にリセットされるが、
		// その 8/24 の記録を削除すると再計算で「8/3 週を最終週とする5週連続」に戻る。
		// 今日(8/30)から見ればとうに途切れているので、表示上は0週・フリーズ未使用で返す。
		// last_recorded_week は DATE カラムのため UTC の 0時 として読み出される点を再現する。
		overrideTimeNow(t, time.Date(2026, 8, 30, 23, 0, 0, 0, time.Local))
		stored := entity.NewUserStreak("user-1", 5, 5, 0, 0, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), time.Now())
		userStreakRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(stored, nil)

		streak, err := u.GetByUserId(context.Background(), "user-1")

		require.NoError(t, err)
		require.Equal(t, 0, streak.CurrentWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
		require.Equal(t, 5, streak.LongestWeeks)
	})

	t.Run("正常系_フリーズ猶予(2週間)を超え、かつフリーズ使用済みなら1週間経過時点でも終了扱い", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
		u := NewStreak(userStreakRepo)

		lastWeek := mondayOf(time.Now()).AddDate(0, 0, -21)
		stored := entity.NewUserStreak("user-1", 2, 2, 1, 0, lastWeek, time.Now())
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

	// user_streaks.last_recorded_week は DATE カラムのため、DBからは UTC の 0時 として
	// 読み出される。現在時刻(ローカル)の月曜 0時 との差を瞬間で取ると 9時間分だけ短くなり、
	// 3週前が「2週前」に見えて猶予内と誤判定していた(記録削除で最終記録週が過去へ戻った
	// 直後に、既に途切れているはずのストリークがフリーズ付きで復活して見える)。
	t.Run("正常系_DB由来のUTC日付でも3週前なら期限切れ", func(t *testing.T) {
		overrideTimeNow(t, time.Date(2026, 8, 30, 23, 0, 0, 0, time.Local)) // 日曜(今週の月曜は 8/24)
		require.True(t, isStreakExpired(time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), 0))
	})

	t.Run("正常系_DB由来のUTC日付でも2週前かつフリーズ満杯なら期限切れ", func(t *testing.T) {
		overrideTimeNow(t, time.Date(2026, 8, 30, 23, 0, 0, 0, time.Local))
		require.True(t, isStreakExpired(time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), StreakMaxFreezeCount))
	})

	t.Run("正常系_DB由来のUTC日付で2週前かつフリーズ空きありなら継続扱い", func(t *testing.T) {
		overrideTimeNow(t, time.Date(2026, 8, 30, 23, 0, 0, 0, time.Local))
		require.False(t, isStreakExpired(time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), 0))
	})
}
