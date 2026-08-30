// notify-weekend-reminder は、今週まだ記録がなく Web Push を購読しているユーザーへ、
// 週末の記録を想起させる通知(アプリ内通知 + push)を作成する定期バッチ(施策 B-2)。
//
// サイト外から週次で呼び戻す最初の装置。B-5(ストリーク途切れ防止 nudge)がアプリ内通知
// だけでは「サイトに来ない人に届かない」問題を、push を配達手段にして解く
// (B1_B2_PUSH_NOTIFICATION_PLAN.md §5.5 / B1_EXECUTION_STEPS.md Step 2-1)。
//
// 判定は usecase.RemindUser が行う:
//   - 記録経験者(user_streaks に行がある)で、今週(月曜以降)まだ記録が無い
//   - 生きている push 購読がある(購読していない人には物理的に届かないため対象外)
//   - 今週まだ週末リマインドを作っていない(同一週の二重送信ガード。cron 多重起動でも安全)
//   - 直近4回連続で未タップかつ記録も無い人は隔週(ISO偶数週)だけ送る
//
// 1ユーザー週2通の上限(週末リマインド + nudge)は push 送出側(usecase.PushNotifier)が守る。
//
// 想定運用: OSのcronから毎週金曜 20:00 JST に起動する(crontab例は config/crontab 参照)。
// VAPID 鍵(.env の VAPID_*)が未設定なら push は送らず、アプリ内通知だけが作られる。
//
// 使い方:
//
//	# 送信せず対象者と件数だけ確認する(デフォルト)
//	go run ./cmd/notify-weekend-reminder
//
//	# 実際に通知を作成し push を送る
//	go run ./cmd/notify-weekend-reminder -dry-run=false
//
//	# 特定ユーザーのみ対象にする(検証用)
//	go run ./cmd/notify-weekend-reminder -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、通知は作成せず対象者の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全対象ユーザー)")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
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

	reminder := usecase.NewWeekendReminder(
		infrastructure.NewUserStreak(db),
		infrastructure.NewPushSubscription(db),
		infrastructure.NewPushDelivery(db),
		infrastructure.NewNotification(db),
		usecase.NewPushNotifier(
			infrastructure.NewPushSubscription(db),
			infrastructure.NewPushDelivery(db),
			pushSender,
		),
	)

	ctx := context.Background()

	var userIds []string
	if *targetUserId != "" {
		userIds = []string{*targetUserId}
	} else {
		userIds, err = findCandidateUserIds(db)
		if err != nil {
			log.Printf("failed to list candidate users: %v\n", err)
			os.Exit(ExitCodeNG)
		}
	}

	if *dryRun {
		log.Printf("[dry-run] checking weekend-reminder targets among %d subscribed users (通知は作成しません)\n", len(userIds))
	} else {
		log.Printf("sending weekend-reminder among %d subscribed users\n", len(userIds))
	}

	sent := 0
	for _, userId := range userIds {
		ok, err := reminder.RemindUser(ctx, userId, *dryRun)
		if err != nil {
			log.Printf("failed to remind user=%s: %v\n", userId, err)
			continue
		}
		if ok {
			sent++
			if *dryRun {
				log.Printf("[dry-run] TARGET user=%s\n", userId)
			} else {
				log.Printf("reminded user=%s\n", userId)
			}
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d users are weekend-reminder targets\n", sent, len(userIds))
	} else {
		log.Printf("completed: reminded %d/%d users\n", sent, len(userIds))
	}

	os.Exit(ExitCodeOK)
}

// findCandidateUserIds は生きている push 購読を持つユーザーを候補として返す。
// 「今週まだ記録が無いか」「今週送信済みか」の最終判定は usecase.RemindUser が行う。
func findCandidateUserIds(db *gorm.DB) ([]string, error) {
	var userIds []string
	tx := db.Table("push_subscriptions").
		Where("revoked_at IS NULL").
		Distinct("user_id").
		Pluck("user_id", &userIds)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return userIds, nil
}
