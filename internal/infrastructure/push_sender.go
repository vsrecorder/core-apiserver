package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/httpclient"
)

// pushTTLSeconds はプッシュサービスが端末オフライン時にメッセージを保持する秒数。
// 週次の想起なので1日持てば十分で、それ以上古い通知が翌日以降に突然届いても価値が無い。
const pushTTLSeconds = 24 * 60 * 60

// pushErrorBodyLimit は失敗時に読み取るレスポンス本文の上限。理由が分かれば十分なので短く切る。
const pushErrorBodyLimit = 512

var errPushSenderDisabled = errors.New("web push sender is disabled: VAPID keys are not configured")

// WebPushSender は標準 Web Push(RFC 8030 + VAPID)の送出器。
// プッシュサービス(ブラウザベンダーのサーバ)へ暗号化したペイロードを POST する。
type WebPushSender struct {
	publicKey  string
	privateKey string
	// subject はプッシュサービスからの連絡先(mailto: または https:)。VAPID の仕様上必須。
	subject string
	client  *http.Client
}

// NewWebPushSender は VAPID 鍵ペアと連絡先から送出器を作る。
// 鍵が未設定でもエラーにせず「無効な送出器」を返し、Enabled() で判定できるようにする
// (鍵が無い環境でもバッチと API が起動できるようにするため)。
func NewWebPushSender(
	publicKey string,
	privateKey string,
	subject string,
) repository.PushSenderInterface {
	return &WebPushSender{
		publicKey:  publicKey,
		privateKey: privateKey,
		subject:    subject,
		client:     &http.Client{Timeout: httpclient.Timeout},
	}
}

func (s *WebPushSender) Enabled() bool {
	return s.publicKey != "" && s.privateKey != "" && s.subject != ""
}

// pushMessage は端末の Service Worker(webapp/public/sw.js)が読む JSON。
// キー名は sw.js 側と一致させること。
type pushMessage struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	URL        string `json:"url"`
	DeliveryId string `json:"deliveryId"`
	Tag        string `json:"tag"`
}

func (s *WebPushSender) Send(
	ctx context.Context,
	subscription *entity.PushSubscription,
	payload *entity.PushPayload,
) (int, error) {
	if !s.Enabled() {
		return 0, errPushSenderDisabled
	}

	body, err := json.Marshal(pushMessage{
		Title:      payload.Title,
		Body:       payload.Body,
		URL:        payload.URL,
		DeliveryId: payload.DeliveryId,
		Tag:        payload.Tag,
	})
	if err != nil {
		logError(ctx, err)
		return 0, err
	}

	resp, err := webpush.SendNotificationWithContext(
		ctx,
		body,
		&webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				P256dh: subscription.P256dh,
				Auth:   subscription.Auth,
			},
		},
		&webpush.Options{
			HTTPClient:      s.client,
			Subscriber:      s.subject,
			TTL:             pushTTLSeconds,
			Urgency:         webpush.UrgencyNormal,
			VAPIDPublicKey:  s.publicKey,
			VAPIDPrivateKey: s.privateKey,
		},
	)
	if err != nil {
		logError(ctx, err)
		return 0, err
	}
	defer resp.Body.Close()

	// 失敗したときは本文を残す。プッシュサービスは拒否の理由を本文に入れてくるため
	// (Apple は {"reason":"BadJwtToken"} のような JSON、FCM は短い文字列)、ここで捨てると
	// 手掛かりが status code だけになり、鍵・subject・JWT のどれが悪いのか切り分けられない。
	// 実際、iOS へ一通も届かない事故の調査で、403 の理由が分からず行き詰まった。
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, pushErrorBodyLimit))

		slog.WarnContext(ctx, "push service rejected the notification",
			slog.Int("status_code", resp.StatusCode),
			slog.String("push_service", endpointHost(subscription.Endpoint)),
			slog.String("platform", subscription.Platform),
			slog.String("response_body", strings.TrimSpace(string(body))),
		)

		return resp.StatusCode, nil
	}

	// 成功時の本文に用は無いが、読み切らないと接続が再利用されない
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

// endpointHost は endpoint からホスト名だけを取り出す。どのプッシュサービスが
// 拒否したのかをログで見分けるために使う(endpoint 全体は購読の識別子なので出さない)。
func endpointHost(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}

	return u.Host
}
