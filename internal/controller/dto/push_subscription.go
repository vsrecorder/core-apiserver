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

// PushSubscriptionCreateResponse は購読登録の結果。
//
// WasRevoked は「その endpoint が失効済みとして残っていた」ことを表す。配信側が
// 404/410・連続失敗で失効させた購読は、同じ endpoint で登録し直しても大抵死んだままなので、
// 端末はこれを見て購読を作り直す。端末には購読オブジェクトが残り、端末だけでは
// 失効に気付けないため、サーバから伝える必要がある。
type PushSubscriptionCreateResponse struct {
	WasRevoked bool `json:"was_revoked"`
}

// PushSubscriptionDeleteRequest は購読解除。endpoint で端末を特定する。
type PushSubscriptionDeleteRequest struct {
	Endpoint string `json:"endpoint"`
}
