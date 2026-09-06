package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// NotificationCategoryStreak は連続記録に関する通知(entity.Notification)のカテゴリ。
// webappのNotificationCategoryと一致させる。
const NotificationCategoryStreak = "streak"

const (
	// streakNudgeLinkUrl は途切れ防止nudgeのリンク先。タップしたらそのまま記録作成へ入れる。
	streakNudgeLinkUrl = "/records/create"

	// streakNudgeTitle は途切れ防止nudgeの見出し。同一週内の二重送信判定(dedup)の
	// キーにもなるため固定文言にする。
	streakNudgeTitle = "連続記録がとぎれそうです"

	// streakNudgeDedupScanLimit は二重送信判定で遡って確認する直近通知の件数。
	// 1ユーザーが1週間で受け取る通知はバッジ達成等を含めても十分小さいため、
	// この件数を見れば「今週すでにnudgeを送ったか」を取りこぼさない。
	streakNudgeDedupScanLimit = 30
)

type StreakNudgeInterface interface {
	// NudgeUser は指定ユーザーの連続記録が「今週記録しないと途切れる瀬戸際」かを判定し、
	// 該当し、かつ今週まだnudgeを送っていなければ途切れ防止のアプリ内通知を1件作成する。
	// dryRun=true の場合は作成対象かどうかだけを返し、通知は作らない。
	// 戻り値の bool は「作成した(dryRunなら作成対象だった)」かどうか。
	NudgeUser(ctx context.Context, userId string, dryRun bool) (bool, error)
}

type StreakNudge struct {
	userStreakRepo   repository.UserStreakInterface
	notificationRepo repository.NotificationInterface
	pushNotifier     PushNotifierInterface
}

func NewStreakNudge(
	userStreakRepo repository.UserStreakInterface,
	notificationRepo repository.NotificationInterface,
	pushNotifier PushNotifierInterface,
) StreakNudgeInterface {
	return &StreakNudge{userStreakRepo, notificationRepo, pushNotifier}
}

func (u *StreakNudge) NudgeUser(ctx context.Context, userId string, dryRun bool) (bool, error) {
	streak, err := u.userStreakRepo.FindByUserId(ctx, userId)
	if err != nil {
		logError(ctx, err)
		if errors.Is(err, apperror.ErrRecordNotFound) {
			// まだ一度も記録していない(=守るべき連続が無い)ユーザーは対象外
			return false, nil
		}
		return false, err
	}

	now := timeNow()

	if !isLastChanceThisWeek(streak.LastRecordedWeek, streak.FreezeUsedCount, now) {
		return false, nil
	}

	already, err := u.alreadyNudgedThisWeek(ctx, userId, now)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if already {
		return false, nil
	}

	if dryRun {
		return true, nil
	}

	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return false, err
	}

	body := fmt.Sprintf("今週まだ記録がありません。1件記録すると%d週連続をキープできます", streak.CurrentWeeks)

	notification := entity.NewNotification(
		id,
		now,
		userId,
		NotificationCategoryStreak,
		streakNudgeTitle,
		body,
		streakNudgeLinkUrl,
	)

	if err := u.notificationRepo.Save(ctx, notification); err != nil {
		logError(ctx, err)
		return false, err
	}

	// B-1: アプリ内通知を作った上で push を撃つ(D2)。宛先は定義上「今週サイトに来ていない人」なので、
	// サイト外へ押し出せる push が無いとこの通知は見られない。push の失敗で通知作成は巻き戻さない
	if _, err := u.pushNotifier.Deliver(ctx, notification, PushCampaignStreakNudge); err != nil {
		logWarn(ctx, err)
	}

	return true, nil
}

// alreadyNudgedThisWeek は、今週(月曜以降)に既に途切れ防止nudgeを送っているかを返す。
// cronの多重起動などで同じ週に2通目を作らないための冪等性ガード。
func (u *StreakNudge) alreadyNudgedThisWeek(ctx context.Context, userId string, now time.Time) (bool, error) {
	thisMonday := mondayOf(now)

	notifications, err := u.notificationRepo.FindByUserId(ctx, userId, streakNudgeDedupScanLimit)
	if err != nil {
		logError(ctx, err)
		return false, err
	}

	for _, n := range notifications {
		// 見出しが nudge 固定文言で、今週(月曜以降)に作られたものが1件でもあれば送信済みとみなす
		if n.Title == streakNudgeTitle && !n.CreatedAt.Before(thisMonday) {
			return true, nil
		}
	}

	return false, nil
}

// isLastChanceThisWeek は「今週記録しなければ連続記録が途切れてしまう最後の週」かどうかを判定する。
// ComputeStreakState / isStreakExpired と同じ週・フリーズの基準(canKeepStreak)を用いる:
//   - 今はまだ連続が生きている(今週記録すれば継続できる)
//   - かつ、今週サボって次週に記録しても、その時にはもう継続できない(空白週数が残りフリーズを超える)
//
// これにより「まだフリーズが残っていて余裕がある人」を煽らず、本当に今週が瀬戸際の人だけに送る。
// 具体的には、最終記録週から今週までの空白週数(gap週-1)が残りフリーズ数とちょうど等しい
// (今週も空けると埋め切れなくなる)ときだけ true。例: フリーズ満杯(残り0)なら先週が最後の人、
// 残り1なら2週前が最後の人、残り2なら3週前が最後の人。
func isLastChanceThisWeek(lastRecordedWeek time.Time, freezeUsedCount int, now time.Time) bool {
	if lastRecordedWeek.IsZero() {
		return false
	}

	// lastRecordedWeek は DATE カラム由来(UTC の 0時)、now はローカル時刻なので、
	// 瞬間の差ではなく暦日ベースで週差を取る(weeksBetween 参照)。
	gapWeeks := weeksBetween(lastRecordedWeek, now)

	// 今週(またはそれ以降の未来週)に既に記録済み → 対象外
	if gapWeeks <= 0 {
		return false
	}

	// 今週記録すれば連続を維持できるか(=今はまだ生きているか)。
	// 維持できないなら既に空白週が残りフリーズを超えて途切れており、送っても手遅れ(復帰施策の領域)。
	if !canKeepStreak(gapWeeks, freezeUsedCount) {
		return false
	}

	// 今週サボって次週(gapWeeks+1)に記録した場合、そのとき連続を維持できるか。
	// 維持できる(=まだ猶予がある)なら今週は瀬戸際ではないので送らない。
	return !canKeepStreak(gapWeeks+1, freezeUsedCount)
}
