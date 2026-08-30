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

// weekendReminderOddWeekNow は 2026-08-28(金) 20:00。週の偶奇(isEvenWeek)は奇数。mondayOf は 08-24(月)。
// weekendReminderEvenWeekNow は 2026-09-04(金) 20:00。週の偶奇(isEvenWeek)は偶数。mondayOf は 08-31(月)。
var (
	weekendReminderOddWeekNow  = time.Date(2026, 8, 28, 20, 0, 0, 0, time.Local)
	weekendReminderEvenWeekNow = time.Date(2026, 9, 4, 20, 0, 0, 0, time.Local)
)

type weekendReminderMocks struct {
	userStreak       *mock_repository.MockUserStreakInterface
	pushSubscription *mock_repository.MockPushSubscriptionInterface
	pushDelivery     *mock_repository.MockPushDeliveryInterface
	notification     *mock_repository.MockNotificationInterface
	pushNotifier     *stubPushNotifier
}

func setup4WeekendReminder(t *testing.T, now time.Time) (*weekendReminderMocks, WeekendReminderInterface) {
	t.Helper()
	overrideTimeNow(t, now)

	mockCtrl := gomock.NewController(t)
	m := &weekendReminderMocks{
		userStreak:       mock_repository.NewMockUserStreakInterface(mockCtrl),
		pushSubscription: mock_repository.NewMockPushSubscriptionInterface(mockCtrl),
		pushDelivery:     mock_repository.NewMockPushDeliveryInterface(mockCtrl),
		notification:     mock_repository.NewMockNotificationInterface(mockCtrl),
		pushNotifier:     &stubPushNotifier{sent: 1},
	}

	return m, NewWeekendReminder(m.userStreak, m.pushSubscription, m.pushDelivery, m.notification, m.pushNotifier)
}

func liveSubscriptions() []*entity.PushSubscription {
	return []*entity.PushSubscription{
		entity.NewPushSubscription("sub-1", time.Now(), "user-1", "https://push.example.com/1", "p", "a", entity.PushPlatformAndroid),
	}
}

// unclickedDeliveries は直近 weeks 週ぶんの未タップの配達ログ(週ごとに端末2台分)を返す。
func unclickedDeliveries(from time.Time, weeks int) []*entity.PushDelivery {
	var deliveries []*entity.PushDelivery
	for i := 0; i < weeks; i++ {
		at := from.AddDate(0, 0, -7*i)
		deliveries = append(deliveries,
			entity.NewPushDelivery("d-a-"+string(rune('0'+i)), at, "user-1", "sub-1", "n", PushCampaignWeekendReminder, entity.PushDeliveryStatusSent, 201),
			entity.NewPushDelivery("d-b-"+string(rune('0'+i)), at, "user-1", "sub-2", "n", PushCampaignWeekendReminder, entity.PushDeliveryStatusSent, 201),
		)
	}
	return deliveries
}

func TestWeekendReminder_RemindUser(t *testing.T) {
	t.Run("正常系_今週まだ記録が無く購読があれば通知を作成してpushする", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)
		lastWeekMonday := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 3, 3, 0, 0, lastWeekMonday, time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(liveSubscriptions(), nil)
		m.notification.EXPECT().FindByUserId(gomock.Any(), "user-1", weekendReminderDedupScanLimit).Return(nil, nil)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignWeekendReminder, weekendReminderRecentScanLimit).Return(nil, nil)

		var saved *entity.Notification
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.True(t, sent)
		// 通知を作った上で、その通知を push で配達する(D2)
		require.Len(t, m.pushNotifier.calls, 1)
		require.Same(t, saved, m.pushNotifier.calls[0].notification)
		require.Equal(t, PushCampaignWeekendReminder, m.pushNotifier.calls[0].campaign)
		require.Equal(t, NotificationCategoryReminder, saved.Category)
		require.Equal(t, weekendReminderTitle, saved.Title)
		require.Equal(t, weekendReminderBody, saved.Body)
		require.Equal(t, weekendReminderLinkUrl, saved.LinkUrl)
		require.Equal(t, weekendReminderOddWeekNow, saved.CreatedAt)
	})

	t.Run("対象外_今週すでに記録済みなら送らない", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)
		thisMonday := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 4, 4, 0, 0, thisMonday, time.Now()), nil)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_購読が無ければ送らない(アプリ内通知だけ作っても届かない)", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 3, 3, 0, 0, time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_今週すでに送信済みなら2通目を作らない", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 3, 3, 0, 0, time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(liveSubscriptions(), nil)
		m.notification.EXPECT().FindByUserId(gomock.Any(), "user-1", weekendReminderDedupScanLimit).Return([]*entity.Notification{
			// 先週のリマインドは妨げにならない
			entity.NewNotification("n-old", weekendReminderOddWeekNow.AddDate(0, 0, -7), "user-1", NotificationCategoryReminder, weekendReminderTitle, "", weekendReminderLinkUrl),
			// 今週のリマインドが1件でもあれば送信済み
			entity.NewNotification("n-1", weekendReminderOddWeekNow.Add(-time.Hour), "user-1", NotificationCategoryReminder, weekendReminderTitle, "", weekendReminderLinkUrl),
		}, nil)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("dry-run_対象でも通知は作成せずtrueを返す", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 3, 3, 0, 0, time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(liveSubscriptions(), nil)
		m.notification.EXPECT().FindByUserId(gomock.Any(), "user-1", weekendReminderDedupScanLimit).Return(nil, nil)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignWeekendReminder, weekendReminderRecentScanLimit).Return(nil, nil)
		// Save も Deliver も呼ばれない

		sent, err := u.RemindUser(context.Background(), "user-1", true)

		require.NoError(t, err)
		require.True(t, sent)
		require.Empty(t, m.pushNotifier.campaigns())
	})

	t.Run("隔週ガード_直近4回未タップかつ4週以上記録が無ければ奇数週は送らない", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)
		// 最終記録は6週前(4週以上前)
		longAgo := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 0, 3, 0, 0, longAgo, time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(liveSubscriptions(), nil)
		m.notification.EXPECT().FindByUserId(gomock.Any(), "user-1", weekendReminderDedupScanLimit).Return(nil, nil)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignWeekendReminder, weekendReminderRecentScanLimit).
			Return(unclickedDeliveries(weekendReminderOddWeekNow.AddDate(0, 0, -7), 4), nil)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("隔週ガード_同じ条件でも偶数週なら送る", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderEvenWeekNow)
		longAgo := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 0, 3, 0, 0, longAgo, time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(liveSubscriptions(), nil)
		m.notification.EXPECT().FindByUserId(gomock.Any(), "user-1", weekendReminderDedupScanLimit).Return(nil, nil)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignWeekendReminder, weekendReminderRecentScanLimit).
			Return(unclickedDeliveries(weekendReminderEvenWeekNow.AddDate(0, 0, -7), 4), nil)
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, []string{PushCampaignWeekendReminder}, m.pushNotifier.campaigns())
	})

	t.Run("隔週ガード_直近4回のどれかをタップしていれば毎週送る", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)
		longAgo := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)

		deliveries := unclickedDeliveries(weekendReminderOddWeekNow.AddDate(0, 0, -7), 4)
		deliveries[3].ClickedAt = deliveries[3].CreatedAt.Add(time.Hour) // 2週前の端末Bでタップ

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 0, 3, 0, 0, longAgo, time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(liveSubscriptions(), nil)
		m.notification.EXPECT().FindByUserId(gomock.Any(), "user-1", weekendReminderDedupScanLimit).Return(nil, nil)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignWeekendReminder, weekendReminderRecentScanLimit).Return(deliveries, nil)
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, []string{PushCampaignWeekendReminder}, m.pushNotifier.campaigns())
	})

	t.Run("隔週ガード_未タップでも直近4週のうちに記録があれば毎週送る", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)
		// 最終記録は2週前(直近4週の中)
		recent := time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 0, 3, 0, 0, recent, time.Now()), nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(liveSubscriptions(), nil)
		m.notification.EXPECT().FindByUserId(gomock.Any(), "user-1", weekendReminderDedupScanLimit).Return(nil, nil)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignWeekendReminder, weekendReminderRecentScanLimit).
			Return(unclickedDeliveries(weekendReminderOddWeekNow.AddDate(0, 0, -7), 4), nil)
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, []string{PushCampaignWeekendReminder}, m.pushNotifier.campaigns())
	})

	t.Run("対象外_ストリーク行が無い(記録経験なし)ユーザーは送らない", func(t *testing.T) {
		m, u := setup4WeekendReminder(t, weekendReminderOddWeekNow)

		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)

		sent, err := u.RemindUser(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.False(t, sent)
	})
}
