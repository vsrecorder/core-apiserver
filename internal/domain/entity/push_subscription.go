package entity

import (
	"time"
)

// 購読端末の種別。iOS はホーム画面に追加した PWA でしか Web Push を受け取れないため、
// 「iOS で届く人」の規模を単独で追えるよう他と分けて持つ(B1_B2_PUSH_NOTIFICATION_PLAN.md §6)。
const (
	PushPlatformIOSPWA  = "ios-pwa"
	PushPlatformAndroid = "android"
	PushPlatformDesktop = "desktop"
)

var pushPlatforms = map[string]struct{}{
	PushPlatformIOSPWA:  {},
	PushPlatformAndroid: {},
	PushPlatformDesktop: {},
}

// NormalizePushPlatform は既知の種別ならそのまま、未知なら空文字を返す。
// クライアントの申告値をそのまま保存すると集計の軸が無限に増えるため、ここで丸める。
func NormalizePushPlatform(platform string) string {
	if _, ok := pushPlatforms[platform]; ok {
		return platform
	}

	return ""
}

// PushSubscription は Web Push の購読。端末ごとに1行で、endpoint が実質のデバイス識別子。
// 1ユーザーが複数端末を持ちうる。RevokedAt がゼロ値でなければ解除・失効済み。
type PushSubscription struct {
	ID            string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RevokedAt     time.Time
	UserId        string
	Endpoint      string
	P256dh        string
	Auth          string
	Platform      string
	FailureCount  int
	LastSuccessAt time.Time
}

func NewPushSubscription(
	id string,
	createdAt time.Time,
	userId string,
	endpoint string,
	p256dh string,
	auth string,
	platform string,
) *PushSubscription {
	return &PushSubscription{
		ID:        id,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		UserId:    userId,
		Endpoint:  endpoint,
		P256dh:    p256dh,
		Auth:      auth,
		Platform:  platform,
	}
}

// IsRevoked は解除・失効済みかを返す。
func (s *PushSubscription) IsRevoked() bool {
	return !s.RevokedAt.IsZero()
}
