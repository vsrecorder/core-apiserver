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

	// weekendReminderQuietAfter は「反応が無い」とみなす直近の配信回数。
	// この回数連続で未クリック、かつその間に記録も無ければ隔週配信に落とす。
	weekendReminderQuietAfter = 4

	// weekendReminderRecentScanLimit は隔週判定で遡る配達ログの件数。
	// 1回の配信で端末数ぶんの行ができるため、回数より多めに読む。
	weekendReminderRecentScanLimit = 20
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

	quiet, err := u.isUnresponsive(ctx, userId, streak, thisMonday)
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

// isUnresponsive は「直近 weekendReminderQuietAfter 回の週末リマインドをすべてタップしておらず、
// その間に記録もしていない」かを返す。配達ログは端末ごとに行ができるため、週(月曜)単位に畳んでから数える。
func (u *WeekendReminder) isUnresponsive(ctx context.Context, userId string, streak *entity.UserStreak, thisMonday time.Time) (bool, error) {
	deliveries, err := u.pushDeliveryRepo.FindRecentByUserIdAndCampaign(
		ctx, userId, PushCampaignWeekendReminder, weekendReminderRecentScanLimit,
	)
	if err != nil {
		return false, err
	}

	weeks := map[time.Time]bool{}
	var order []time.Time
	for _, d := range deliveries {
		week := mondayOf(d.CreatedAt)
		if _, ok := weeks[week]; !ok {
			weeks[week] = false
			order = append(order, week)
		}
		if d.IsClicked() {
			weeks[week] = true
		}
	}

	if len(order) < weekendReminderQuietAfter {
		return false, nil
	}
	for _, week := range order[:weekendReminderQuietAfter] {
		if weeks[week] {
			return false, nil
		}
	}

	// 直近の配信期間中に記録があれば「反応が無い」とはみなさない(タップせずに直接開いた可能性)
	quietSince := thisMonday.AddDate(0, 0, -7*weekendReminderQuietAfter)
	return streak.LastRecordedWeek.Before(quietSince), nil
}

// weekParityEpoch は週の偶奇を数える起点(1970-01-05 は月曜)。
// ISO 週番号だと年またぎで 53→1 と奇数が続き、隔週対象者が2週連続で飛ぶことがあるため、
// 起点からの経過週数で数える。
var weekParityEpoch = time.Date(1970, 1, 5, 0, 0, 0, 0, time.UTC)

// isEvenWeek は monday(週の月曜0時)が起点から偶数週目かを返す。隔週配信の週を決めるのに使う。
// ローカル時刻の月曜0時を UTC の暦日に読み替えて日数を数えるため、タイムゾーンに依存しない。
func isEvenWeek(monday time.Time) bool {
	day := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	weeks := int(day.Sub(weekParityEpoch).Hours()/24) / 7
	return weeks%2 == 0
}
