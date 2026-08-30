package validation

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/ratelimit"
)

// 購読登録の連打を抑えるレート制限。通常は購読時に1回、以後は1日1回の再同期しか呼ばれない。
var pushSubscribeLimiterByUID = ratelimit.New(30, time.Hour)

const (
	// pushEndpointMaxLength はプッシュサービスの URL の上限。実際は数百文字だが余裕を持たせる。
	pushEndpointMaxLength = 2048
	// pushKeyMaxLength は p256dh / auth(base64url)の上限。スキーマの VARCHAR(255) に合わせる。
	pushKeyMaxLength = 255
)

// isValidPushEndpoint はプッシュサービスの URL として最低限の形かを見る。
//
// 配信バッチはここに保存された URL へサーバ側から POST するため、任意の URL を通すと
// 内部ネットワークへのリクエストに使われうる(SSRF)。ペイロードは購読者の鍵で暗号化されて
// いるので情報は漏れないが、宛先として明らかにプッシュサービスでないもの
// (http・IP アドレス直指定・localhost・ドットの無いホスト名)は弾く。
// ブラウザベンダーごとにホスト名が異なる(FCM / Apple / Mozilla / WNS など)ため、
// ホストの許可リストは持たない。
func isValidPushEndpoint(endpoint string) bool {
	if !strings.HasPrefix(endpoint, "https://") || len(endpoint) > pushEndpointMaxLength {
		return false
	}

	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return false
	}
	if net.ParseIP(host) != nil || !strings.Contains(host, ".") {
		return false
	}

	return true
}

func isValidPushKey(key string) bool {
	return key != "" && len(key) <= pushKeyMaxLength
}

func PushSubscriptionCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.PushSubscriptionCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if !isValidPushEndpoint(req.Endpoint) || !isValidPushKey(req.Keys.P256dh) || !isValidPushKey(req.Keys.Auth) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		// 未知の platform は弾かずに空文字へ丸める(クライアントの先行デプロイで購読が落ちないように)
		req.Platform = entity.NormalizePushPlatform(req.Platform)

		if !pushSubscribeLimiterByUID.Allow(helper.GetUID(ctx)) {
			apierror.ErrTooManyRequests.JSON(ctx)
			return
		}

		helper.SetPushSubscriptionCreateRequest(ctx, req)
	}
}

func PushSubscriptionDeleteMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.PushSubscriptionDeleteRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if !isValidPushEndpoint(req.Endpoint) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetPushSubscriptionDeleteRequest(ctx, req)
	}
}
