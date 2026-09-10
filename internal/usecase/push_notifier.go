package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"

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
	// PushCampaignEnvNews は P-2 の代替、記録ゼロの週の環境ニュース(月曜 08:00・購読者のみ)。
	PushCampaignEnvNews = "env_news"
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
// 月曜の weekly_report(先週記録した人への配当)と env_news(記録ゼロの週の代替)は数えない。
// これにより1人あたり最大でも 月(レポート or 環境ニュース)・金(週末)・日(nudge) の3通に収まる。
// 反応の無い人への間引き(isPushUnresponsive)は env_news と weekend_reminder が各自で行う。
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

	// sentOnce は、このプロセスがプッシュサービスに1件でも受理されたかを表す。
	// 403(VAPID の資格情報が拒否された)を「サーバの鍵設定ミス」と「その購読だけが古い
	// 公開鍵で作られている」のどちらとして扱うかの切り分けに使う。鍵設定が誤っていれば
	// 全端末が403になるためこのフラグは永久に立たず、購読には一切触れない。
	// 鍵はプロセス起動時に .env から読むので、「差し替える前の成功」がここに残ることもない
	// (バッチは実行ごとに新しいプロセス、APIサーバは再デプロイで作り直される)。
	sentOnce atomic.Bool
}

func NewPushNotifier(
	subscriptionRepo repository.PushSubscriptionInterface,
	deliveryRepo repository.PushDeliveryInterface,
	sender repository.PushSenderInterface,
) PushNotifierInterface {
	return &PushNotifier{
		subscriptionRepo: subscriptionRepo,
		deliveryRepo:     deliveryRepo,
		sender:           sender,
	}
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
		// 他の端末へは受理されている状況での403は、鍵設定ではなくこの購読が古い公開鍵で
		// 作られていることを意味する。再購読されるまで永久に成功しないため失効として扱う
		// (失効させないと、死んだ購読へ毎日送り続けてERRORを出し続けることになる)。
		if status == entity.PushDeliveryStatusFailed && u.isStaleCredentialRejection(statusCode) {
			status = entity.PushDeliveryStatusExpired

			slog.WarnContext(ctx, "push subscription is bound to an outdated VAPID key: revoking",
				slog.Int("status_code", statusCode),
				slog.String("subscription_id", subscription.ID),
				slog.String("user_id", notification.UserId),
				slog.String("push_service", pushServiceHost(subscription.Endpoint)),
				slog.String("platform", subscription.Platform),
			)
		}

		if err := u.deliveryRepo.UpdateResult(ctx, id, status, statusCode); err != nil {
			// 結果が残らなくても送出はしているので続行する(計測が欠けるだけ)
			logError(ctx, err)
		}

		switch status {
		case entity.PushDeliveryStatusSent:
			sent++
			// 受理された実績を残し、以後の403を「鍵設定ミス」ではなく
			// 「その購読が古い」と断定できるようにする
			u.sentOnce.Store(true)

			if err := u.subscriptionRepo.MarkSuccess(ctx, subscription.ID, now); err != nil {
				logError(ctx, err)
			}

		case entity.PushDeliveryStatusExpired:
			// 404/410(プッシュサービスが購読を無効と判断した)と、他端末へ届いている
			// 状況での403(古い鍵で作られた購読)。以後この端末には送らない
			if err := u.subscriptionRepo.Revoke(ctx, subscription.ID, now); err != nil {
				logError(ctx, err)
			}

		default:
			if !countsAsSubscriptionFailure(statusCode, sendErr) {
				// 400/401/413 と、まだ1件も受理されていない状況での403は、購読ではなく
				// 送信側(VAPID 鍵・subject・ペイロード)の問題。購読の失敗回数に数えると
				// 数週間で全購読が失効し、許諾を取り直すことになる。
				// 購読には触れず、調査が必要なエラーとして残す
				slog.ErrorContext(ctx, "push service rejected the request: check VAPID keys / subject / payload",
					slog.Int("status_code", statusCode),
					slog.String("subscription_id", subscription.ID),
					slog.String("user_id", notification.UserId),
					slog.String("push_service", pushServiceHost(subscription.Endpoint)),
					slog.String("platform", subscription.Platform),
				)
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

// isStaleCredentialRejection は 403 を「その購読が古い VAPID 公開鍵で作られている」と
// 断定してよいかを返す。同じプロセスで既に他の端末へ受理されていれば、鍵・subject・
// ペイロードは正しいと分かるため、拒否の原因はその購読側にしかない。
//
// 逆にまだ1件も受理されていないうちは、鍵設定を誤った直後である可能性が残る。そこで
// 失効させると全ユーザーに許諾を取り直させることになるため、断定せず購読を守る。
func (u *PushNotifier) isStaleCredentialRejection(statusCode int) bool {
	return statusCode == http.StatusForbidden && u.sentOnce.Load()
}

// pushServiceHost は endpoint からホスト名だけを取り出す。
// endpoint はそれ自体がその端末へ push を送れてしまう秘密情報なので、ログには残さない。
func pushServiceHost(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "(unknown)"
	}

	return parsed.Host
}

// countsAsSubscriptionFailure は、その失敗を「端末(購読)側の問題」として失敗回数に数えるかを返す。
// 通信失敗・5xx・429(プッシュサービス側の一時的な不調)は数え、連続すれば購読を失効させる。
// それ以外の 4xx(400/401/413 など)は送信側の設定ミスなので数えない。403 は失効として
// 扱うか設定ミスとして扱うかが呼び出し側で決まるため、ここでは数えない側に倒す。
func countsAsSubscriptionFailure(statusCode int, sendErr error) bool {
	if sendErr != nil {
		return true
	}

	return statusCode >= 500 || statusCode == http.StatusTooManyRequests || statusCode == http.StatusRequestTimeout
}
