package entity

import (
	"time"
)

// 配達ログの状態。プッシュサービスへの送出結果をそのまま表す。
// 端末に「届いた」「タップされた」は DeliveredAt / ClickedAt で別に持つ。
const (
	// PushDeliveryStatusPending は送出前に採番・記録した直後の状態。端末が到達を報告する前に
	// 行が存在するようにするため、送出の前に作る。送出結果で sent/failed/expired へ更新される。
	PushDeliveryStatusPending = "pending"
	// PushDeliveryStatusSent はプッシュサービスが受理した(2xx)。
	PushDeliveryStatusSent = "sent"
	// PushDeliveryStatusFailed は受理されなかった(5xx・通信失敗など)。購読はまだ生きている。
	PushDeliveryStatusFailed = "failed"
	// PushDeliveryStatusExpired は購読が無効(404/410)。購読は失効させる。
	PushDeliveryStatusExpired = "expired"
)

// PushDelivery は push の配達ログ。notifications が「通知の実体」であるのに対し、
// こちらは「配達というチャネル固有の事象」で、1通知が複数端末へ配達されうるため 1:N になる。
// 「許諾率 × 到達率 × 反応率」の分解(B1_B2_PUSH_NOTIFICATION_PLAN.md §6)を測るために持つ。
type PushDelivery struct {
	ID             string
	CreatedAt      time.Time
	UserId         string
	SubscriptionId string
	NotificationId string
	Campaign       string
	Status         string
	StatusCode     int
	// DeliveredAt は端末の Service Worker が push を受け取った時刻(ゼロ値なら未到達)。
	DeliveredAt time.Time
	// ClickedAt は通知がタップされた時刻(ゼロ値なら未タップ)。
	ClickedAt time.Time
}

func NewPushDelivery(
	id string,
	createdAt time.Time,
	userId string,
	subscriptionId string,
	notificationId string,
	campaign string,
	status string,
	statusCode int,
) *PushDelivery {
	return &PushDelivery{
		ID:             id,
		CreatedAt:      createdAt,
		UserId:         userId,
		SubscriptionId: subscriptionId,
		NotificationId: notificationId,
		Campaign:       campaign,
		Status:         status,
		StatusCode:     statusCode,
	}
}

// IsClicked は通知がタップされたかを返す。
func (d *PushDelivery) IsClicked() bool {
	return !d.ClickedAt.IsZero()
}

// PushPayload は端末の Service Worker へ渡す push の中身。
// webapp/public/sw.js の push ハンドラが読む JSON と1対1で対応する。
type PushPayload struct {
	Title string
	Body  string
	// URL は通知タップ時に開くサイト内パス(notifications.link_url)。
	URL string
	// DeliveryId は配達ログのID。端末側が到達・タップの計測に使う。
	DeliveryId string
	// Tag はブラウザ側で同じ種類の通知を置き換えるためのキー(キャンペーン名)。
	Tag string
}
