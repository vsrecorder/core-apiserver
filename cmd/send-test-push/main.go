/*
 * 指定したユーザーの「生きている購読」へ、テストの push を1通ずつ送る。
 *
 * 通常のバッチ(notify-*)は「今週まだ記録していない」「今週まだ送っていない」といった
 * 送信条件を持つため、調べたいときに限って一通も飛ばない。このコマンドは条件判定を
 * 通さず、購読へ直接送ってプッシュサービスの応答だけを見る。
 *
 * 届かない原因の切り分けに使う。特に 403 は、応答本文の理由(Apple なら
 * {"reason":"BadJwtToken"} など)まで見ないと鍵・subject・JWT のどれが悪いのか分からない。
 * 理由は infrastructure の WebPushSender が WARN で出す。
 *
 * 副作用を残さないため、アプリ内通知(notifications)も配達ログ(push_deliveries)も作らない。
 * 購読の失効・失敗回数の更新も行わない(調査でユーザーの購読状態を変えてしまわないように)。
 * そのぶん端末側の到達計測(deliveryId)も無いので、届いたかどうかは端末で直接確認する。
 *
 *   ./bin/send-test-push -user-id=xxxxx
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

const appName = "send-test-push"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	targetUserId := flag.String("user-id", "", "送信先のユーザーID(必須)")
	title := flag.String("title", "テスト送信", "通知のタイトル")
	body := flag.String("body", "プッシュ通知の到達確認です", "通知の本文")
	flag.Parse()

	// 全ユーザーへ一斉にテスト通知を投げてしまわないよう、対象の指定を必須にする。
	if *targetUserId == "" {
		slog.Error("-user-id is required")
		os.Exit(ExitCodeNG)
	}

	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", logging.Err(err))
	}

	db, err := postgres.NewDB(
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_USER_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		slog.Error("failed to connect database", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	pushSender := infrastructure.NewWebPushSender(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_SUBJECT"),
	)
	if !pushSender.Enabled() {
		slog.Error("web push is disabled: VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY / VAPID_SUBJECT are not set")
		os.Exit(ExitCodeNG)
	}

	ctx := context.Background()

	subscriptions, err := infrastructure.NewPushSubscription(db).FindLiveByUserId(ctx, *targetUserId)
	if err != nil {
		slog.Error("failed to find live subscriptions", slog.String("user_id", *targetUserId), logging.Err(err))
		os.Exit(ExitCodeNG)
	}
	if len(subscriptions) == 0 {
		slog.Warn("no live subscription", slog.String("user_id", *targetUserId))
		os.Exit(ExitCodeOK)
	}

	slog.Info("sending test push",
		slog.String("user_id", *targetUserId),
		slog.Int("subscriptions", len(subscriptions)),
	)

	failed := 0
	for _, subscription := range subscriptions {
		// DeliveryId は空。配達ログを作らないので、端末に到達報告させる先が無い
		status, err := pushSender.Send(ctx, subscription, &entity.PushPayload{
			Title: *title,
			Body:  *body,
			URL:   "/",
			Tag:   "test",
		})
		if err != nil {
			failed++
			slog.Error("failed to send",
				slog.String("subscription_id", subscription.ID),
				slog.String("platform", subscription.Platform),
				logging.Err(err),
			)
			continue
		}

		// 2xx 以外は理由を WebPushSender が WARN で出している。ここでは結果だけまとめる
		if status >= http.StatusBadRequest {
			failed++
		}

		fmt.Printf("subscription=%s platform=%-8s status=%d\n", subscription.ID, subscription.Platform, status)
	}

	slog.Info("completed",
		slog.String("user_id", *targetUserId),
		slog.Int("subscriptions", len(subscriptions)),
		slog.Int("failed", failed),
	)

	if failed > 0 {
		os.Exit(ExitCodeNG)
	}

	os.Exit(ExitCodeOK)
}
