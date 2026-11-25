// notify-deck-code-post-likes は、みんなの公開デッキの投稿へ前日に付いたいいねを
// 投稿ごとに1通にまとめて投稿者へ通知する日次バッチ(config/crontab から毎日 21:00 JST に実行)。
//
// 通知の対象は「前日(ローカル時刻の暦日)に付いた、投稿者以外のいいね」がある公開中の投稿。
// 取り下げ済み・運営が非表示にした投稿は対象にしない。.env の OFFICIAL_USER_ID(運営の公式
// アカウント)が押したいいねも数えない。cmd/auto-like-deck-code-posts が全投稿へ自動で
// 付けるものなので、通知すると毎日同じ文面が投稿者全員へ届いてしまうため。通知のリンク先に対象日を含めて
// 重複判定に使うため、同じ投稿・同じ日の通知は二重に作らず、再実行しても安全。
//
//	go run ./cmd/notify-deck-code-post-likes                                  # 前日分の対象を確認するだけ(dry-run)
//	go run ./cmd/notify-deck-code-post-likes -dry-run=false                   # 前日分を通知する
//	go run ./cmd/notify-deck-code-post-likes -date=2026-09-03 -dry-run=false  # 対象日を指定する
//	go run ./cmd/notify-deck-code-post-likes -user-id=xxxx -dry-run=false     # 特定の投稿者だけ通知する
//
// 投稿ごとの処理は最初の失敗で打ち切り、非 0 で終える(通知は重複判定があるので、再実行すれば
// 残りから続く)。-notify-slack を付けると、止まったことを SLACK_WEBHOOK_URL へ知らせる。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/batchreport"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
	"github.com/vsrecorder/core-apiserver/internal/slack"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "notify-deck-code-post-likes"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	// ログは cmd/core-apiserver と同じJSON形式に揃える。これを呼ばないと slog の
	// 既定ハンドラ(テキスト)のままになり、usecase 層のログから layer やソース位置が
	// 落ちて、cron のログだけ他と違う読み方を強いられる。
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	dryRun := flag.Bool("dry-run", true, "true の場合、通知は作成せず対象の件数を確認するだけにする")
	dateFlag := flag.String("date", "", "対象日(YYYY-MM-DD)。未指定なら前日")
	userId := flag.String("user-id", "", "指定した場合、その投稿者宛ての通知だけを対象にする")
	notifySlack := flag.Bool("notify-slack", false, "true の場合、途中で止まったら SLACK_WEBHOOK_URL へ通知する")
	flag.Parse()

	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", logging.Err(err))
	}

	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	// 通知するつもりで通知できない状態は、黙って流して終わるより早く気付きたい
	if *notifySlack && slackWebhookURL == "" {
		slog.Error("SLACK_WEBHOOK_URL is not set: -notify-slack requires it")
		os.Exit(ExitCodeNG)
	}

	// 止まったことを知らせる先(-notify-slack を付けたときだけ)
	var notifyFailure batchreport.Notifier
	if *notifySlack {
		notifyFailure = func(message string) error { return slack.Notify(slackWebhookURL, message) }
	}

	day := time.Now().AddDate(0, 0, -1)
	if *dateFlag != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, *dateFlag, time.Local)
		if err != nil {
			slog.Error("invalid -date", slog.String("date", *dateFlag), logging.Err(err))
			os.Exit(ExitCodeNG)
		}
		day = parsed
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
		os.Exit(batchreport.Abort(appName, "データベースに接続できませんでした", notifyFailure))
	}

	pushSender := infrastructure.NewWebPushSender(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_SUBJECT"),
	)
	if !pushSender.Enabled() {
		slog.Warn("web push is disabled: VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY / VAPID_SUBJECT are not set. in-app notifications only")
	}
	pushNotifier := usecase.NewPushNotifier(
		infrastructure.NewPushSubscription(db),
		infrastructure.NewPushDelivery(db),
		pushSender,
	)

	// 運営の公式アカウント(cmd/auto-like-deck-code-posts)が自動で押したいいねは通知しない。
	// 全投稿に付くため、通知すると毎日同じ文面が投稿者全員へ届き、人が押したいいねの
	// 知らせが埋もれる。未設定なら除外せず、これまで通り全てのいいねを数える。
	officialUserId := os.Getenv("OFFICIAL_USER_ID")

	notifier := usecase.NewDeckCodePostLikeNotifier(
		infrastructure.NewDeckCodePost(db),
		infrastructure.NewNotification(db),
		pushNotifier,
		officialUserId,
	)

	// dry-run と対象は属性で出す(メッセージを分岐させると grep の条件が増える)。
	// target_user_id が空文字なら全ユーザーが対象。
	dayLabel := day.Format(time.DateOnly)
	batchAttrs := []any{
		slog.String("date", dayLabel),
		slog.String("target_user_id", *userId),
		slog.String("excluded_liker_user_id", officialUserId),
		slog.Bool("dry_run", *dryRun),
	}
	slog.Info("notifying like digests", batchAttrs...)

	count, err := notifier.NotifyDay(context.Background(), day, *userId, *dryRun)
	if err != nil {
		slog.Error("failed to notify like digests",
			append(batchAttrs, slog.Int("notified_posts", count), logging.Err(err))...)
		os.Exit(batchreport.Abort(
			appName,
			fmt.Sprintf("対象日 %s のいいねのまとめ通知が、%d 件進んだところで止まりました", dayLabel, count),
			notifyFailure,
		))
	}

	slog.Info("completed", append(batchAttrs, slog.Int("notified_posts", count))...)

	os.Exit(ExitCodeOK)
}
