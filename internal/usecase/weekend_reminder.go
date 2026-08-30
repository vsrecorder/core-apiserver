package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// NotificationCategoryReminder は週末リマインドの通知カテゴリ。
	// webapp 側の NotificationCategory / ベルのアイコン対応表と一致させる。
	NotificationCategoryReminder = "reminder"

	// 文言は B1_B2_PUSH_NOTIFICATION_PLAN.md §5.5 のとおり。見出しは同一週の二重送信判定の
	// キーにもなるため固定文言にする。リンク先は最短フォーム(/records/quick)。
	weekendReminderTitle   = "今週末、対戦の予定は？"
	weekendReminderBody    = "帰り道の3タップで記録できます。先週の戦績もすぐ見返せます"
	weekendReminderLinkUrl = "/records/quick"

	// weekendReminderDedupScanLimit は二重送信判定で遡って確認する直近通知の件数(B-5 と同じ考え方)。
	weekendReminderDedupScanLimit = 30
)

type WeekendReminderInterface interface {
	// RemindUser は「今週まだ記録が無く、push の購読がある記録経験者」へ、週末の記録を想起させる
	// 通知を1件作成して push する。同じ週に既に送っていれば作らない(冪等)。
	// dryRun=true の場合は作成対象かどうかだけを返し、通知は作らない。
	// 戻り値の bool は「作成した(dryRunなら作成対象だった)」かどうか。
	RemindUser(ctx context.Context, userId string, dryRun bool) (bool, error)
}

type WeekendReminder struct {
	userStreakRepo       repository.UserStreakInterface
	pushSubscriptionRepo repository.PushSubscriptionInterface
	pushDeliveryRepo     repository.PushDeliveryInterface
	notificationRepo     repository.NotificationInterface
	pushNotifier         PushNotifierInterface
}

func NewWeekendReminder(
	userStreakRepo repository.UserStreakInterface,
	pushSubscriptionRepo repository.PushSubscriptionInterface,
	pushDeliveryRepo repository.PushDeliveryInterface,
	notificationRepo repository.NotificationInterface,
	pushNotifier PushNotifierInterface,
) WeekendReminderInterface {
	return &WeekendReminder{
		userStreakRepo:       userStreakRepo,
		pushSubscriptionRepo: pushSubscriptionRepo,
		pushDeliveryRepo:     pushDeliveryRepo,
		notificationRepo:     notificationRepo,
		pushNotifier:         pushNotifier,
	}
}

func (u *WeekendReminder) RemindUser(ctx context.Context, userId string, dryRun bool) (bool, error) {
	streak, err := u.userStreakRepo.FindByUserId(ctx, userId)
	if err != nil {
		logError(ctx, err)
		if errors.Is(err, apperror.ErrRecordNotFound) {
			// まだ一度も記録していない人は「週末に記録を思い出させる」対象ではない
			return false, nil
		}
		return false, err
	}

	now := timeNow()
	thisMonday := mondayOf(now)

	// 今週(またはそれ以降)に既に記録済みなら想起は不要
	if !streak.LastRecordedWeek.Before(thisMonday) {
		return false, nil
	}

	// 購読が無い人には物理的に届かない。アプリ内通知だけ作っても B-5 と同じ問題
	// (サイトに来ないと見えない)を繰り返すので、購読者だけに絞る
	subscriptions, err := u.pushSubscriptionRepo.FindLiveByUserId(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if len(subscriptions) == 0 {
		return false, nil
	}

	already, err := u.alreadyRemindedThisWeek(ctx, userId, thisMonday)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if already {
		return false, nil
	}

	quiet, err := isPushUnresponsive(ctx, u.pushDeliveryRepo, userId, PushCampaignWeekendReminder, streak.LastRecordedWeek, thisMonday)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	// 反応の無い人には隔週だけ送る。毎週撃つのは許諾取り消しの最短経路
	if quiet && !isEvenWeek(thisMonday) {
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

	notification := entity.NewNotification(
		id,
		now,
		userId,
		NotificationCategoryReminder,
		weekendReminderTitle,
		weekendReminderBody,
		weekendReminderLinkUrl,
	)

	if err := u.notificationRepo.Save(ctx, notification); err != nil {
		logError(ctx, err)
		return false, err
	}

	// D2: アプリ内通知を作った上で push を撃つ。push の失敗で通知作成は巻き戻さない
	if _, err := u.pushNotifier.Deliver(ctx, notification, PushCampaignWeekendReminder); err != nil {
		logWarn(ctx, err)
	}

	return true, nil
}

// alreadyRemindedThisWeek は、今週(月曜以降)に既に週末リマインドを作っているかを返す。
// 判定先を notifications にしているのは、push が上限や鍵未設定で送られなかった週でも
// 同じ週に2通目のアプリ内通知を作らないため。
func (u *WeekendReminder) alreadyRemindedThisWeek(ctx context.Context, userId string, thisMonday time.Time) (bool, error) {
	notifications, err := u.notificationRepo.FindByUserId(ctx, userId, weekendReminderDedupScanLimit)
	if err != nil {
		return false, err
	}

	for _, n := range notifications {
		if n.Category == NotificationCategoryReminder && n.Title == weekendReminderTitle && !n.CreatedAt.Before(thisMonday) {
			return true, nil
		}
	}

	return false, nil
}
