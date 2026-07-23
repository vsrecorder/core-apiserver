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

const (
	// streakNudgeLinkUrl は途切れ防止nudgeのリンク先。タップしたらそのまま記録作成へ入れる。
	streakNudgeLinkUrl = "/records/create"

	// streakNudgeTitle は途切れ防止nudgeの見出し。達成通知("ストリークを継続中です")と
	// 区別でき、同一週内の二重送信判定(dedup)のキーにもなるため固定文言にする。
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
}

func NewStreakNudge(
	userStreakRepo repository.UserStreakInterface,
	notificationRepo repository.NotificationInterface,
) StreakNudgeInterface {
	return &StreakNudge{userStreakRepo, notificationRepo}
}

func (u *StreakNudge) NudgeUser(ctx context.Context, userId string, dryRun bool) (bool, error) {
	streak, err := u.userStreakRepo.FindByUserId(ctx, userId)
	if err != nil {
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
		return false, err
	}

	return true, nil
}

// alreadyNudgedThisWeek は、今週(月曜以降)に既に途切れ防止nudgeを送っているかを返す。
// cronの多重起動などで同じ週に2通目を作らないための冪等性ガード。
func (u *StreakNudge) alreadyNudgedThisWeek(ctx context.Context, userId string, now time.Time) (bool, error) {
	thisMonday := mondayOf(now)

	notifications, err := u.notificationRepo.FindByUserId(ctx, userId, streakNudgeDedupScanLimit)
	if err != nil {
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
// updateStreak / isStreakExpired と同じ週・フリーズ猶予の基準を用いる:
//   - 今はまだ連続が生きている(今週記録すれば継続できる)
//   - かつ、今週サボって次週に記録しても、その時にはもう継続できない(フリーズ猶予を超える)
//
// これにより「まだフリーズ猶予が残っていて余裕がある人」を煽らず、本当に今週が瀬戸際の人だけに送る。
// 具体的には (gap週=1 かつ フリーズ満杯) または (gap週=2 かつ フリーズ空きあり) のときだけ true。
func isLastChanceThisWeek(lastRecordedWeek time.Time, freezeUsedCount int, now time.Time) bool {
	if lastRecordedWeek.IsZero() {
		return false
	}

	thisMonday := mondayOf(now)
	gapWeeks := int(thisMonday.Sub(lastRecordedWeek).Hours()/24) / 7

	// 今週(またはそれ以降の未来週)に既に記録済み → 対象外
	if gapWeeks <= 0 {
		return false
	}

	// 今週記録すれば連続を維持できるか(=今はまだ生きているか)。
	// 維持できないなら既にフリーズ猶予を超えて途切れており、送っても手遅れ(復帰施策の領域)。
	if !canKeepStreak(gapWeeks, freezeUsedCount) {
		return false
	}

	// 今週サボって次週(gapWeeks+1)に記録した場合、そのとき連続を維持できるか。
	// 維持できる(=まだ猶予がある)なら今週は瀬戸際ではないので送らない。
	return !canKeepStreak(gapWeeks+1, freezeUsedCount)
}

// canKeepStreak は、最後の記録から gapWeeks 週あいた時点で記録した場合に、
// updateStreak の基準で連続記録を維持できる(リセットされない)かを返す。
func canKeepStreak(gapWeeks int, freezeUsedCount int) bool {
	if gapWeeks == 1 {
		return true
	}
	return gapWeeks <= streakFreezeMaxGapWeeks && freezeUsedCount < StreakMaxFreezeCount
}
