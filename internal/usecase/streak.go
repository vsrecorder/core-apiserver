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
		logError(ctx, err)
		if errors.Is(err, apperror.ErrRecordNotFound) {
			// まだ一度も記録していないユーザーは0件のストリークとして返す
			return entity.NewUserStreak(userId, 0, 0, 0, 0, time.Time{}, time.Time{}), nil
		}

		return nil, err
	}

	return projectStreakToNow(streak), nil
}

// projectStreakToNow は user_streaks の保存値を「今日から見た状態」に読み替えて返す。
//
// 保存値は records から作り直した「最後に記録した時点」の状態で、その後は記録の
// 作成・削除・更新まで誰も更新しない。時間が経つだけで変わるべきものは、参照のたびに
// ここで今週との差分を見て反映する。DB上の値自体は書き換えない(次の記録作成・削除・
// 更新時の再計算が records から同じ結論を出すため、書き換える必要がない)。
//
//   - 猶予を超えて途切れている → 連続0週・フリーズ未使用(最長記録は残す)
//   - 先週が未記録(最終記録週から2週あき)でフリーズに空きがある → その空き週で使う
//     フリーズを消費済みとして返す。ComputeStreakState は記録と記録の「間」の空き週しか
//     数えないため、次の記録が来るまで保存値には現れないが、次に記録した時点で必ず
//     消費される(ComputeStreakState が freezeUsedCount++、回復進捗0 にする)。表示だけ
//     「まだ残っている」と見せると、サボった直後は減らず記録した瞬間に減るという
//     分かりにくい動きになり、nudge(canKeepStreak)が「フリーズを使う前提」で数えている
//     見立てとも食い違うため、ここで先に消費済みとして扱う。
//     空き週に後から対戦日を遡って記録した場合は再計算で消費が戻るが、それは保存値の
//     再計算と同じ挙動なので矛盾しない。
func projectStreakToNow(streak *entity.UserStreak) *entity.UserStreak {
	if isStreakExpired(streak.LastRecordedWeek, streak.FreezeUsedCount) {
		return entity.NewUserStreak(streak.UserId, 0, streak.LongestWeeks, 0, 0, streak.LastRecordedWeek, streak.UpdatedAt)
	}

	// 途切れていない かつ 2週以上あいている = 猶予内でフリーズに空きがある状態。
	// 1回の空きで消費するフリーズは ComputeStreakState と同じく1つ(空き週数ぶんではない)。
	if hasPendingFreeze(streak.LastRecordedWeek) {
		return entity.NewUserStreak(streak.UserId, streak.CurrentWeeks, streak.LongestWeeks, streak.FreezeUsedCount+1, 0, streak.LastRecordedWeek, streak.UpdatedAt)
	}

	return streak
}

// hasPendingFreeze は、最終記録週から今週までに未記録の週(先週)があるかを返す。
// 途切れているかどうかは見ないため、isStreakExpired が false のときにだけ使うこと。
func hasPendingFreeze(lastRecordedWeek time.Time) bool {
	if lastRecordedWeek.IsZero() {
		return false
	}

	return weeksBetween(lastRecordedWeek, timeNow()) > 1
}

// isStreakExpired は、今週の時点で lastRecordedWeek からの記録が既に途切れているかを判定する。
// updateStreak の「フリーズで継続扱いにできるか(diffWeeks<=streakFreezeMaxGapWeeks かつ
// フリーズ未使用)」という条件をそのまま流用し、新規記録が来ていない状態でも同じ基準で
// 「今日記録したとしたら継続扱いになるか」を評価する。
func isStreakExpired(lastRecordedWeek time.Time, freezeUsedCount int) bool {
	if lastRecordedWeek.IsZero() {
		return false
	}

	// lastRecordedWeek は DATE カラム由来(UTC の 0時)、timeNow() はローカル時刻なので、
	// 瞬間の差ではなく暦日ベースで週差を取る(weeksBetween 参照)。
	diffWeeks := weeksBetween(lastRecordedWeek, timeNow())

	if diffWeeks <= 1 {
		return false
	}
	if diffWeeks <= streakFreezeMaxGapWeeks && freezeUsedCount < StreakMaxFreezeCount {
		return false
	}

	return true
}
