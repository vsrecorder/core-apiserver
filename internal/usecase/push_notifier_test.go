package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// pushNotifierFixedNow は 2026-08-28(金) 20:00。mondayOf は 2026-08-24(月)。
var pushNotifierFixedNow = time.Date(2026, 8, 28, 20, 0, 0, 0, time.Local)

type pushNotifierMocks struct {
	subscription *mock_repository.MockPushSubscriptionInterface
	delivery     *mock_repository.MockPushDeliveryInterface
	sender       *mock_repository.MockPushSenderInterface
}

func setup4PushNotifier(t *testing.T) (*pushNotifierMocks, PushNotifierInterface) {
	t.Helper()
	overrideTimeNow(t, pushNotifierFixedNow)

	mockCtrl := gomock.NewController(t)
	m := &pushNotifierMocks{
		subscription: mock_repository.NewMockPushSubscriptionInterface(mockCtrl),
		delivery:     mock_repository.NewMockPushDeliveryInterface(mockCtrl),
		sender:       mock_repository.NewMockPushSenderInterface(mockCtrl),
	}

	return m, NewPushNotifier(m.subscription, m.delivery, m.sender)
}

func newTestPushSubscription(id string, failureCount int) *entity.PushSubscription {
	s := entity.NewPushSubscription(id, pushNotifierFixedNow.AddDate(0, 0, -30), "user-1", "https://push.example.com/"+id, "p256dh", "auth", entity.PushPlatformAndroid)
	s.FailureCount = failureCount
	return s
}

// expectLiveAndUnderCap は「送出器が有効・購読あり・週上限未満」までの共通の期待を張る。
func (m *pushNotifierMocks) expectLiveAndUnderCap(subs []*entity.PushSubscription, countThisWeek int) {
	thisMonday := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)
	m.sender.EXPECT().Enabled().Return(true)
	m.subscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(subs, nil)
	m.delivery.EXPECT().CountNotificationsByUserIdAndCampaignsSince(gomock.Any(), "user-1", pushCampaignsCountedForCap, thisMonday).Return(countThisWeek, nil)
}

func TestPushNotifier_Deliver(t *testing.T) {
	notification := entity.NewNotification("n-1", pushNotifierFixedNow, "user-1", NotificationCategoryReminder, "今週末、対戦の予定は？", "本文", "/records/quick")

	t.Run("正常系_生きている購読すべてへ送り配達ログをpendingで作ってから結果を書く", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{
			newTestPushSubscription("sub-1", 0),
			newTestPushSubscription("sub-2", 0),
		}, 0)

		var saved []*entity.PushDelivery
		var payloads []*entity.PushPayload
		gomock.InOrder(
			// 端末1: 配達ログ作成 → 送出 → 結果
			m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, d *entity.PushDelivery) error {
				saved = append(saved, d)
				return nil
			}),
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *entity.PushSubscription, p *entity.PushPayload) (int, error) {
				payloads = append(payloads, p)
				return 201, nil
			}),
			m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil),
			m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil),
			// 端末2
			m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, d *entity.PushDelivery) error {
				saved = append(saved, d)
				return nil
			}),
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *entity.PushSubscription, p *entity.PushPayload) (int, error) {
				payloads = append(payloads, p)
				return 201, nil
			}),
			m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil),
			m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-2", pushNotifierFixedNow).Return(nil),
		)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 2, sent)
		require.Len(t, saved, 2)
		require.Len(t, payloads, 2)
		require.Equal(t, "今週末、対戦の予定は？", payloads[0].Title)
		require.Equal(t, "/records/quick", payloads[0].URL)
		require.Equal(t, PushCampaignWeekendReminder, payloads[0].Tag)
		// 配達ログは送出前に pending で作られ、その id がペイロードの deliveryId になる
		require.Equal(t, entity.PushDeliveryStatusPending, saved[0].Status)
		require.Equal(t, payloads[0].DeliveryId, saved[0].ID)
		require.Equal(t, "n-1", saved[0].NotificationId)
		require.Equal(t, "sub-1", saved[0].SubscriptionId)
		require.Equal(t, "sub-2", saved[1].SubscriptionId)
	})

	t.Run("正常系_404_410なら購読を失効させexpiredで記録する", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, 0)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(410, nil)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusExpired, 410).Return(nil)
		m.subscription.EXPECT().Revoke(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 0, sent)
	})

	t.Run("正常系_5xxならfailure_countを増やし閾値に達したら失効させる", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{
			newTestPushSubscription("sub-1", 0),                           // 1回目の失敗 → まだ生かす
			newTestPushSubscription("sub-2", pushRevokeAfterFailures-1), // 今回で閾値 → 失効
		}, 0)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(2)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(503, nil).Times(2)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusFailed, 503).Return(nil).Times(2)
		m.subscription.EXPECT().IncrementFailure(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)
		m.subscription.EXPECT().IncrementFailure(gomock.Any(), "sub-2", pushNotifierFixedNow).Return(nil)
		m.subscription.EXPECT().Revoke(gomock.Any(), "sub-2", pushNotifierFixedNow).Return(nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 0, sent)
	})

	t.Run("正常系_通信失敗もfailedとして記録し失敗回数を増やす", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, 0)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, errors.New("dial tcp: timeout"))
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusFailed, 0).Return(nil)
		m.subscription.EXPECT().IncrementFailure(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignStreakNudge)

		require.NoError(t, err)
		require.Equal(t, 0, sent)
	})

	t.Run("正常系_401_403_400は送信側の設定ミスなので購読の失敗回数に数えない", func(t *testing.T) {
		for _, code := range []int{400, 401, 403, 413} {
			m, u := setup4PushNotifier(t)
			// 閾値の直前でも失効させない
			m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", pushRevokeAfterFailures-1)}, 0)
			m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(code, nil)
			m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusFailed, code).Return(nil)
			// IncrementFailure も Revoke も呼ばれない

			sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

			require.NoError(t, err, code)
			require.Equal(t, 0, sent, code)
		}
	})

	t.Run("正常系_配達ログが作れなかった端末には送らず次の端末へ進む", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{
			newTestPushSubscription("sub-1", 0),
			newTestPushSubscription("sub-2", 0),
		}, 0)
		gomock.InOrder(
			m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db down")),
			// sub-1 には Send しない
			m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil),
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil),
			m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil),
			m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-2", pushNotifierFixedNow).Return(nil),
		)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 1, sent)
	})

	t.Run("正常系_結果の書き込みに失敗しても送出済みとして数える", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, 0)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(errors.New("db down"))
		m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 1, sent)
	})

	t.Run("対象外_週2通の上限に達していれば送らない", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, pushWeeklyCap)
		// Save も Send も呼ばれない

		sent, err := u.Deliver(context.Background(), notification, PushCampaignStreakNudge)

		require.NoError(t, err)
		require.Equal(t, 0, sent)
	})

	t.Run("正常系_上限の1つ手前なら送る(境界)", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, pushWeeklyCap-1)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil)
		m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignStreakNudge)

		require.NoError(t, err)
		require.Equal(t, 1, sent)
	})

	t.Run("正常系_週次レポートは上限を数えずに送る", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.sender.EXPECT().Enabled().Return(true)
		m.subscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, nil)
		// CountNotificationsByUserIdAndCampaignsSince は呼ばれない
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil)
		m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeeklyReport)

		require.NoError(t, err)
		require.Equal(t, 1, sent)
	})

	t.Run("対象外_購読が無ければ何もしない", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.sender.EXPECT().Enabled().Return(true)
		m.subscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 0, sent)
	})

	t.Run("対象外_送出器が無効(鍵未設定)なら購読も引かずに何もしない", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.sender.EXPECT().Enabled().Return(false)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 0, sent)
	})

	t.Run("異常系_購読の取得に失敗したらエラーを返す", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.sender.EXPECT().Enabled().Return(true)
		m.subscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, errors.New("db down"))

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.Error(t, err)
		require.Equal(t, 0, sent)
	})
}
