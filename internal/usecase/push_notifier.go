package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// push のキャンペーン名。push_deliveries.campaign と、端末側で通知を置き換える tag に使う。
const (
	// PushCampaignStreakNudge は B-5 ストリーク途切れ防止 nudge(日曜 20:00)。
	PushCampaignStreakNudge = "streak_nudge"
	// PushCampaignWeeklyReport は P-2 週次バトルレポート(月曜 08:00)。
	PushCampaignWeeklyReport = "weekly_report"
	// PushCampaignWeekendReminder は B-2 週末リマインド(金曜 20:00)。
	PushCampaignWeekendReminder = "weekend_reminder"
)

const (
	// pushWeeklyCap は1ユーザーが1週間に受け取る想起系 push の上限(B1_B2_PUSH_NOTIFICATION_PLAN.md D5)。
	// 「うるさい」は許諾取り消しに直結し、取り消しは回復不能なので効果より優先する。
	pushWeeklyCap = 2

	// pushRevokeAfterFailures は連続でこの回数失敗した購読を失効させる閾値。
	// 死んだ端末に永久に撃ち続けないため。成功すれば failure_count は0に戻る。
	pushRevokeAfterFailures = 5
)

// pushCampaignsCountedForCap は週あたり上限に数えるキャンペーン。
// 週次レポート(weekly_report)は「先週記録した人への配当」であって想起ではないため数えない。
// これにより1人あたり最大でも 月(レポート)・金(週末)・日(nudge) の3通に収まる。
var pushCampaignsCountedForCap = []string{PushCampaignWeekendReminder, PushCampaignStreakNudge}

type PushNotifierInterface interface {
	// Deliver は作成済みのアプリ内通知を、そのユーザーの生きている購読すべてへ push で配達し、
	// プッシュサービスが受理した端末数を返す。
	//
	// push は「アプリ内通知の配達手段」であり(D2)、送出に失敗しても通知は残す。
	// そのため送出の失敗は戻り値の error にせず、配達ログ(push_deliveries)に記録して
	// 購読の失効・失敗回数の更新だけを行う。error を返すのは購読の取得など送出前の失敗のみ。
	Deliver(
		ctx context.Context,
		notification *entity.Notification,
		campaign string,
	) (int, error)
}

type PushNotifier struct {
	subscriptionRepo repository.PushSubscriptionInterface
	deliveryRepo     repository.PushDeliveryInterface
	sender           repository.PushSenderInterface
}

func NewPushNotifier(
	subscriptionRepo repository.PushSubscriptionInterface,
	deliveryRepo repository.PushDeliveryInterface,
	sender repository.PushSenderInterface,
) PushNotifierInterface {
	return &PushNotifier{subscriptionRepo, deliveryRepo, sender}
}

func (u *PushNotifier) Deliver(
	ctx context.Context,
	notification *entity.Notification,
	campaign string,
) (int, error) {
	// 鍵未設定の環境では何もしない(アプリ内通知だけが残る)
	if !u.sender.Enabled() {
		return 0, nil
	}

	subscriptions, err := u.subscriptionRepo.FindLiveByUserId(ctx, notification.UserId)
	if err != nil {
		logError(ctx, err)
		return 0, err
	}
	if len(subscriptions) == 0 {
		return 0, nil
	}

	now := timeNow()

	if countsTowardWeeklyCap(campaign) {
		count, err := u.deliveryRepo.CountNotificationsByUserIdAndCampaignsSince(
			ctx, notification.UserId, pushCampaignsCountedForCap, mondayOf(now),
		)
		if err != nil {
			logError(ctx, err)
			return 0, err
		}
		if count >= pushWeeklyCap {
			slog.InfoContext(ctx, "push skipped: weekly cap reached",
				slog.String("campaign", campaign),
				slog.Int("count", count),
			)
			return 0, nil
		}
	}

	sent := 0
	for _, subscription := range subscriptions {
		id, err := generateId()
		if err != nil {
			logError(ctx, err)
			continue
		}

		// 配達ログは送出の前に pending で作る。ペイロードに載せる deliveryId を端末が
		// 到達報告に使うため、送出後に作ると「届いたのに行が無い」窓ができる。
		// 行が作れなければその端末には送らない(ID だけが出回るのを避ける)。
		delivery := entity.NewPushDelivery(
			id, now, notification.UserId, subscription.ID, notification.ID, campaign, entity.PushDeliveryStatusPending, 0,
		)
		if err := u.deliveryRepo.Save(ctx, delivery); err != nil {
			logError(ctx, err)
			continue
		}

		statusCode, sendErr := u.sender.Send(ctx, subscription, &entity.PushPayload{
			Title:      notification.Title,
			Body:       notification.Body,
			URL:        notification.LinkUrl,
			DeliveryId: id,
			Tag:        campaign,
		})

		status := pushDeliveryStatus(statusCode, sendErr)
		if err := u.deliveryRepo.UpdateResult(ctx, id, status, statusCode); err != nil {
			// 結果が残らなくても送出はしているので続行する(計測が欠けるだけ)
			logError(ctx, err)
		}

		switch status {
		case entity.PushDeliveryStatusSent:
			sent++
			if err := u.subscriptionRepo.MarkSuccess(ctx, subscription.ID, now); err != nil {
				logError(ctx, err)
			}

		case entity.PushDeliveryStatusExpired:
			// 404/410 はプッシュサービスが購読を無効と判断した合図。以後この端末には送らない
			if err := u.subscriptionRepo.Revoke(ctx, subscription.ID, now); err != nil {
				logError(ctx, err)
			}

		default:
			if !countsAsSubscriptionFailure(statusCode, sendErr) {
				// 400/401/403/413 などは購読ではなく送信側(VAPID 鍵・subject・ペイロード)の問題。
				// 購読の失敗回数に数えると数週間で全購読が失効し、許諾を取り直すことになる。
				// 購読には触れず、調査が必要なエラーとして残す
				logError(ctx, fmt.Errorf("push service rejected the request (status %d): check VAPID keys / subject / payload", statusCode))
				continue
			}

			if sendErr != nil {
				logWarn(ctx, sendErr)
			} else {
				logWarn(ctx, fmt.Errorf("push service responded with status %d", statusCode))
			}
			if err := u.subscriptionRepo.IncrementFailure(ctx, subscription.ID, now); err != nil {
				logError(ctx, err)
			}
			// 取得時点の failure_count に今回の1回を足して判定する(同一バッチ内の並行更新は無い前提)
			if subscription.FailureCount+1 >= pushRevokeAfterFailures {
				if err := u.subscriptionRepo.Revoke(ctx, subscription.ID, now); err != nil {
					logError(ctx, err)
				}
			}
		}
	}

	return sent, nil
}

func countsTowardWeeklyCap(campaign string) bool {
	for _, c := range pushCampaignsCountedForCap {
		if c == campaign {
			return true
		}
	}

	return false
}

// pushDeliveryStatus はプッシュサービスの応答を配達ログの状態へ落とす。
func pushDeliveryStatus(statusCode int, sendErr error) string {
	switch {
	case sendErr == nil && statusCode >= 200 && statusCode < 300:
		return entity.PushDeliveryStatusSent
	case statusCode == http.StatusNotFound || statusCode == http.StatusGone:
		return entity.PushDeliveryStatusExpired
	default:
		return entity.PushDeliveryStatusFailed
	}
}

// countsAsSubscriptionFailure は、その失敗を「端末(購読)側の問題」として失敗回数に数えるかを返す。
// 通信失敗・5xx・429(プッシュサービス側の一時的な不調)は数え、連続すれば購読を失効させる。
// それ以外の 4xx(400/401/403/413 など)は送信側の設定ミスなので数えない。
func countsAsSubscriptionFailure(statusCode int, sendErr error) bool {
	if sendErr != nil {
		return true
	}

	return statusCode >= 500 || statusCode == http.StatusTooManyRequests || statusCode == http.StatusRequestTimeout
}
