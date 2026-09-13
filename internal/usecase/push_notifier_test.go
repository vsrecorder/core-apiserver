package usecase

import (
	"context"
	"errors"
	"fmt"
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
			newTestPushSubscription("sub-1", 0),                         // 1回目の失敗 → まだ生かす
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

	t.Run("正常系_400_401_413とまだ1件も受理されていないうちの403は購読の失敗回数に数えない", func(t *testing.T) {
		// 403 は鍵設定を誤った直後にも全端末で起きる。1件も受理されていないうちに失効させると
		// 全ユーザーの許諾を取り直すことになるため、この段階では購読に触れない
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

	t.Run("正常系_他端末へ受理されている状況の403は古い鍵の購読としてexpiredで失効させる", func(t *testing.T) {
		m, u := setup4PushNotifier(t)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{
			newTestPushSubscription("sub-1", 0), // 受理される → 鍵・subject・ペイロードは正しい
			newTestPushSubscription("sub-2", 0), // 403 → この購読だけが古い公開鍵で作られている
		}, 0)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(2)
		gomock.InOrder(
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil),
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(403, nil),
		)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil)
		m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusExpired, 403).Return(nil)
		m.subscription.EXPECT().Revoke(gomock.Any(), "sub-2", pushNotifierFixedNow).Return(nil)
		// 失敗回数には数えない(死んだ購読は数える前に失効している)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 1, sent)
	})

	t.Run("正常系_同じプロセスの後続の配達でも受理の実績を引き継いで403を失効させる", func(t *testing.T) {
		// バッチは1プロセスで多数のユーザーを回す。先に別のユーザーへ届いていれば
		// 鍵設定が正しいことは分かっているので、その後の403は購読側の問題と断定できる
		m, u := setup4PushNotifier(t)

		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, 0)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil)
		m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)
		require.NoError(t, err)
		require.Equal(t, 1, sent)

		// 2通目(週上限は weekly_report では数えない)
		m.sender.EXPECT().Enabled().Return(true)
		m.subscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").
			Return([]*entity.PushSubscription{newTestPushSubscription("sub-2", 0)}, nil)
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(403, nil)
		m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusExpired, 403).Return(nil)
		m.subscription.EXPECT().Revoke(gomock.Any(), "sub-2", pushNotifierFixedNow).Return(nil)

		sent, err = u.Deliver(context.Background(), notification, PushCampaignWeeklyReport)
		require.NoError(t, err)
		require.Equal(t, 0, sent)
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

// setup4PushNotifierWithHoldout は、ホールドアウト比率を指定した送出器を作る。
func setup4PushNotifierWithHoldout(t *testing.T, ratio float64) (*pushNotifierMocks, PushNotifierInterface) {
	t.Helper()
	overrideTimeNow(t, pushNotifierFixedNow)

	mockCtrl := gomock.NewController(t)
	m := &pushNotifierMocks{
		subscription: mock_repository.NewMockPushSubscriptionInterface(mockCtrl),
		delivery:     mock_repository.NewMockPushDeliveryInterface(mockCtrl),
		sender:       mock_repository.NewMockPushSenderInterface(mockCtrl),
	}

	return m, NewPushNotifier(m.subscription, m.delivery, m.sender, WithHoldoutRatio(ratio))
}

func TestPushNotifier_Holdout(t *testing.T) {
	notification := entity.NewNotification("n-1", pushNotifierFixedNow, "user-1", NotificationCategoryReminder, "今週末、対戦の予定は？", "本文", "/records/quick")

	t.Run("比率1相当_ホールドアウト群には送らずholdoutの配達ログだけを残す", func(t *testing.T) {
		// 0.9999 は「全ユーザーがホールドアウトに入る」ことを保証する最大の有効値
		// (1 以上は設定ミスとして 0 に丸められるため、実験の上限はこの値になる)。
		m, u := setup4PushNotifierWithHoldout(t, 0.9999)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, 0)

		var saved *entity.PushDelivery
		m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, d *entity.PushDelivery) error {
			saved = d
			return nil
		})
		// Send は一度も呼ばれない(EXPECT を張らないので、呼ばれたらテストが落ちる)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 0, sent)
		require.NotNil(t, saved)
		require.Equal(t, entity.PushDeliveryStatusHoldout, saved.Status)
		require.Equal(t, PushCampaignWeekendReminder, saved.Campaign)
		require.Equal(t, "user-1", saved.UserId)
		require.Equal(t, "sub-1", saved.SubscriptionId)
	})

	t.Run("比率0_実験しない設定なら全員へ送る", func(t *testing.T) {
		m, u := setup4PushNotifierWithHoldout(t, 0)
		m.expectLiveAndUnderCap([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, 0)

		gomock.InOrder(
			m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil),
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil),
			m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil),
			m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil),
		)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeekendReminder)

		require.NoError(t, err)
		require.Equal(t, 1, sent)
	})

	t.Run("対象外キャンペーン_週次レポートは実験しないので必ず送る", func(t *testing.T) {
		m, u := setup4PushNotifierWithHoldout(t, 0.9999)
		// weekly_report は週上限に数えないため、カウントの問い合わせ自体が起きない
		m.sender.EXPECT().Enabled().Return(true)
		m.subscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return([]*entity.PushSubscription{newTestPushSubscription("sub-1", 0)}, nil)

		gomock.InOrder(
			m.delivery.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil),
			m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(201, nil),
			m.delivery.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), entity.PushDeliveryStatusSent, 201).Return(nil),
			m.subscription.EXPECT().MarkSuccess(gomock.Any(), "sub-1", pushNotifierFixedNow).Return(nil),
		)

		sent, err := u.Deliver(context.Background(), notification, PushCampaignWeeklyReport)

		require.NoError(t, err)
		require.Equal(t, 1, sent)
	})
}

func TestPushNotifier_isHoldout(t *testing.T) {
	overrideTimeNow(t, pushNotifierFixedNow)
	notifier := &PushNotifier{holdoutRatio: 0.5}

	t.Run("同じ週なら何度呼んでも同じ群_かつB-2とB-5で群が揃う", func(t *testing.T) {
		first := notifier.isHoldout("user-1", PushCampaignWeekendReminder, pushNotifierFixedNow)

		require.Equal(t, first, notifier.isHoldout("user-1", PushCampaignWeekendReminder, pushNotifierFixedNow))
		// 同一週の日曜(B-5)でも月曜が同じなので群は変わらない
		sunday := pushNotifierFixedNow.AddDate(0, 0, 2)
		require.Equal(t, first, notifier.isHoldout("user-1", PushCampaignStreakNudge, sunday))
	})

	t.Run("比率0_誰もホールドアウトにしない", func(t *testing.T) {
		none := &PushNotifier{holdoutRatio: 0}

		for _, userId := range []string{"user-1", "user-2", "user-3"} {
			require.False(t, none.isHoldout(userId, PushCampaignWeekendReminder, pushNotifierFixedNow))
		}
	})

	t.Run("比率05_多数のユーザーでおおむね半分に割れる", func(t *testing.T) {
		held := 0
		const n = 1000
		for i := range n {
			if notifier.isHoldout(fmt.Sprintf("user-%d", i), PushCampaignWeekendReminder, pushNotifierFixedNow) {
				held++
			}
		}

		// 決定的なハッシュなので毎回同じ値になる。偏りが致命的でないことだけを見る
		require.Greater(t, held, n*4/10)
		require.Less(t, held, n*6/10)
	})
}

func TestParseHoldoutRatio(t *testing.T) {
	// 設定ミスは「実験しない(0)」に倒す。通知が誰にも届かないほうが被害が大きいため
	for _, tt := range []struct {
		raw  string
		want float64
	}{
		{"", 0},
		{"0", 0},
		{"0.5", 0.5},
		{"1", 1},
		{"abc", 0},
	} {
		require.Equal(t, tt.want, ParseHoldoutRatio(tt.raw), "raw=%q", tt.raw)
	}
}

func TestWithHoldoutRatio(t *testing.T) {
	// 範囲外は 0 に丸める(1 以上を許すと全員が想起 push を受け取れなくなる)
	for _, tt := range []struct {
		ratio float64
		want  float64
	}{
		{-0.1, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 0},
		{1.5, 0},
	} {
		n := &PushNotifier{}
		WithHoldoutRatio(tt.ratio)(n)
		require.Equal(t, tt.want, n.holdoutRatio, "ratio=%v", tt.ratio)
	}
}
