package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type StreakInterface interface {
	GetByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserStreak, error)
}

type Streak struct {
	repository repository.UserStreakInterface
}

func NewStreak(
	repository repository.UserStreakInterface,
) StreakInterface {
	return &Streak{repository}
}

func (u *Streak) GetByUserId(
	ctx context.Context,
	userId string,
) (*entity.UserStreak, error) {
	streak, err := u.repository.FindByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			// まだ一度も記録していないユーザーは0件のストリークとして返す
			return entity.NewUserStreak(userId, 0, 0, 0, 0, time.Time{}, time.Time{}), nil
		}

		return nil, err
	}

	// user_streaks は記録の作成・削除時にしか更新されないため、直近の記録から
	// フリーズ猶予を超えて時間が経過している場合、DB上の値は「最後に書き込まれた時点では
	// 正しかったが、今日から見るとすでに途切れているはずの」古い値のままになっている。
	// 新規記録を作らない限り誰も再計算しないため、参照のたびにここで今週との差分を見て
	// 表示上のストリークを終了扱いにする。DB上の値自体は書き換えない(新規記録作成時は
	// updateStreak が本来のLastRecordedWeekを基準に正しく判定するため)。
	if isStreakExpired(streak.LastRecordedWeek, streak.FreezeUsedCount) {
		return entity.NewUserStreak(userId, 0, streak.LongestWeeks, 0, 0, streak.LastRecordedWeek, streak.UpdatedAt), nil
	}

	return streak, nil
}

// isStreakExpired は、今週の時点で lastRecordedWeek からの記録が既に途切れているかを判定する。
// updateStreak の「フリーズで継続扱いにできるか(diffWeeks<=streakFreezeMaxGapWeeks かつ
// フリーズ未使用)」という条件をそのまま流用し、新規記録が来ていない状態でも同じ基準で
// 「今日記録したとしたら継続扱いになるか」を評価する。
func isStreakExpired(lastRecordedWeek time.Time, freezeUsedCount int) bool {
	if lastRecordedWeek.IsZero() {
		return false
	}

	diffDays := int(mondayOf(timeNow()).Sub(lastRecordedWeek).Hours() / 24)
	diffWeeks := diffDays / 7

	if diffWeeks <= 1 {
		return false
	}
	if diffWeeks <= streakFreezeMaxGapWeeks && freezeUsedCount < StreakMaxFreezeCount {
		return false
	}

	return true
}
