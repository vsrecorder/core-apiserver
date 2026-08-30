package dto

// PushSubscriptionKeysRequest は PushSubscription.toJSON() の keys に対応する。
type PushSubscriptionKeysRequest struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// PushSubscriptionCreateRequest は端末の購読登録。ブラウザの PushSubscription.toJSON() に
// platform(ios-pwa / android / desktop)を添えた形。
type PushSubscriptionCreateRequest struct {
	Endpoint string                      `json:"endpoint"`
	Keys     PushSubscriptionKeysRequest `json:"keys"`
	Platform string                      `json:"platform"`
}

// PushSubscriptionDeleteRequest は購読解除。endpoint で端末を特定する。
type PushSubscriptionDeleteRequest struct {
	Endpoint string `json:"endpoint"`
}
