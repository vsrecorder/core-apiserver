// notify-deck-code-post-likes は、みんなの公開デッキの投稿へ前日に付いたいいねを
// 投稿ごとに1通にまとめて投稿者へ通知する日次バッチ(config/crontab から毎朝実行)。
//
// 通知の対象は「前日(ローカル時刻の暦日)に付いた、投稿者以外のいいね」がある公開中の投稿。
// 取り下げ済み・運営が非表示にした投稿は対象にしない。通知のリンク先に対象日を含めて
// 重複判定に使うため、同じ投稿・同じ日の通知は二重に作らず、再実行しても安全。
//
//	go run ./cmd/notify-deck-code-post-likes                                  # 前日分の対象を確認するだけ(dry-run)
//	go run ./cmd/notify-deck-code-post-likes -dry-run=false                   # 前日分を通知する
//	go run ./cmd/notify-deck-code-post-likes -date=2026-09-03 -dry-run=false  # 対象日を指定する
//	go run ./cmd/notify-deck-code-post-likes -user-id=xxxx -dry-run=false     # 特定の投稿者だけ通知する
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、通知は作成せず対象の件数を確認するだけにする")
	dateFlag := flag.String("date", "", "対象日(YYYY-MM-DD)。未指定なら前日")
	userId := flag.String("user-id", "", "指定した場合、その投稿者宛ての通知だけを対象にする")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	day := time.Now().AddDate(0, 0, -1)
	if *dateFlag != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, *dateFlag, time.Local)
		if err != nil {
			log.Printf("invalid -date: %v\n", err)
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
		log.Printf("failed to connect database: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	pushSender := infrastructure.NewWebPushSender(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_SUBJECT"),
	)
	if !pushSender.Enabled() {
		log.Printf("WARN: web push is disabled (VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY / VAPID_SUBJECT are not set). in-app notifications only\n")
	}
	pushNotifier := usecase.NewPushNotifier(
		infrastructure.NewPushSubscription(db),
		infrastructure.NewPushDelivery(db),
		pushSender,
	)

	notifier := usecase.NewDeckCodePostLikeNotifier(
		infrastructure.NewDeckCodePost(db),
		infrastructure.NewNotification(db),
		pushNotifier,
	)

	dayLabel := day.Format(time.DateOnly)
	target := "all users"
	if *userId != "" {
		target = "user " + *userId
	}
	if *dryRun {
		log.Printf("[dry-run] checking like digests for %s (%s). no notifications will be created\n", dayLabel, target)
	} else {
		log.Printf("notifying like digests for %s (%s)\n", dayLabel, target)
	}

	count, err := notifier.NotifyDay(context.Background(), day, *userId, *dryRun)
	if err != nil {
		log.Printf("failed to notify like digests for %s: %v (created=%d)\n", dayLabel, err, count)
		os.Exit(ExitCodeNG)
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d posts would be notified for %s\n", count, dayLabel)
	} else {
		log.Printf("completed: notified %d posts for %s\n", count, dayLabel)
	}

	os.Exit(ExitCodeOK)
}
