package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// PushSenderInterface は Web Push の送出そのもの。プッシュサービス(ブラウザベンダー)への
// 外部通信をインタフェース越しに扱い、usecase のテストで実送信しないようにする
// (deck_asset.go の S3 と同じ作法)。
type PushSenderInterface interface {
	// Enabled は VAPID 鍵が設定され送出できる状態かを返す。
	// 鍵が無い環境(開発機・鍵未配布のデプロイ)でもバッチとAPIが壊れないよう、
	// 呼び出し側はこれを見て送出をスキップする。
	Enabled() bool

	// Send は購読先へ payload を送り、プッシュサービスの HTTP ステータスを返す。
	// 送出自体ができなかった(通信失敗など)場合は 0 と err を返す。
	// 4xx/5xx はエラーではなくステータスとして返し、購読を失効させるかは呼び出し側が判断する。
	Send(
		ctx context.Context,
		subscription *entity.PushSubscription,
		payload *entity.PushPayload,
	) (int, error)
}
