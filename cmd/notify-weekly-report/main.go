// notify-weekly-report は、先週(月〜日)に1戦以上記録したユーザーへ、先週のバトルレポート
// (/users/report/weeks/{先週の月曜}) へ誘導するアプリ内通知を作成する定期バッチ。
//
// 「記録すると資産が貯まり、貯まると見返したくなる」ループの後半を、ユーザー任せにせず
// プロダクト側から毎週起動することを狙う(pmf-plan-2026-08-27.md 施策P-2 /
// P2_WEEKLY_REPORT_PLAN.md)。記録の無い週は配当が無いので送らない(想起は B-5 / B-2 の領域)。
//
// 判定は usecase.NotifyUser が行う: 対象週の戦績を集計して 1戦以上あれば作成、
// 同じ週の通知が既にあれば作らない。週をキーにしているため cron の多重起動でも、
// 同じ -week を指定して再実行しても二重には作らない。
//
// 想定運用: OSのcronから毎週月曜朝に起動する(crontab例は config/crontab 参照)。
// B-1(Web Push)導入後は、アプリ内通知を作った上で購読端末へ push も送る。
// .env の VAPID_* が未設定なら push は送らずアプリ内通知だけになる。
//
// 使い方:
//
//	# 送信せず対象者と件数だけ確認する(デフォルト。対象週は先週)
//	go run ./cmd/notify-weekly-report
//
//	# 実際に通知を作成する
//	go run ./cmd/notify-weekly-report -dry-run=false
//
//	# 対象週を指定する(週内の任意日でよい。過去週のバックフィルや検証用)
//	go run ./cmd/notify-weekly-report -week=2026-08-17
//
//	# 特定ユーザーのみ対象にする(検証用)
//	go run ./cmd/notify-weekly-report -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

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

const weekDateLayout = "2006-01-02"

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、通知は作成せず対象者の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全対象ユーザー)")
	weekFlag := flag.String("week", "", "対象週(週内の任意日 YYYY-MM-DD)。未指定なら先週")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	monday, err := resolveTargetMonday(*weekFlag, time.Now())
	if err != nil {
		log.Printf("invalid -week: %v\n", err)
		os.Exit(ExitCodeNG)
	}
	week := monday.Format(weekDateLayout)
	fromDate, toDate := monday, monday.AddDate(0, 0, 7)

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

	notifier := usecase.NewWeeklyReportNotifier(
		infrastructure.NewUserStat(db),
		infrastructure.NewDeckUsageStat(db),
		infrastructure.NewNotification(db),
		pushNotifier,
	)

	ctx := context.Background()

	var userIds []string
	if *targetUserId != "" {
		userIds = []string{*targetUserId}
	} else {
		userIds, err = findCandidateUserIds(db, fromDate, toDate)
		if err != nil {
			log.Printf("failed to list candidate users: %v\n", err)
			os.Exit(ExitCodeNG)
		}
	}

	if *dryRun {
		log.Printf("[dry-run] checking weekly-report targets among %d users for week=%s (通知は作成しません)\n", len(userIds), week)
	} else {
		log.Printf("sending weekly-report among %d users for week=%s\n", len(userIds), week)
	}

	sent := 0
	for _, userId := range userIds {
		ok, err := notifier.NotifyUser(ctx, userId, week, *dryRun)
		if err != nil {
			log.Printf("failed to notify user=%s week=%s: %v\n", userId, week, err)
			continue
		}
		if ok {
			sent++
			if *dryRun {
				log.Printf("[dry-run] TARGET user=%s week=%s\n", userId, week)
			} else {
				log.Printf("notified user=%s week=%s\n", userId, week)
			}
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d users are weekly-report targets (week=%s)\n", sent, len(userIds), week)
	} else {
		log.Printf("completed: notified %d/%d users (week=%s)\n", sent, len(userIds), week)
	}

	os.Exit(ExitCodeOK)
}

// resolveTargetMonday は対象週の月曜0時(ローカル時刻)を返す。
// week が空なら「先週」(now が属する週の1つ前)、指定があればその日が属する週に正規化する。
func resolveTargetMonday(week string, now time.Time) (time.Time, error) {
	if week == "" {
		return mondayOf(now).AddDate(0, 0, -7), nil
	}

	t, err := time.ParseInLocation(weekDateLayout, week, now.Location())
	if err != nil {
		return time.Time{}, err
	}

	return mondayOf(t), nil
}

// mondayOf は t が属する週(月曜始まり)の月曜0時を返す。usecase/week.go の weekRange と同じ基準。
func mondayOf(t time.Time) time.Time {
	// time.Weekday は日曜=0 ... 土曜=6 なので (weekday+6)%7 で月曜からの経過日数へ変換する
	offset := (int(t.Weekday()) + 6) % 7

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, -offset)
}

// findCandidateUserIds は対象週に開催日(event_date)を持つ記録があるユーザーを候補として返す。
// 「1戦以上あるか」「集計対象外(ignore_stats_flg)を除いて残るか」の最終判定は
// usecase.NotifyUser が戦績集計で行うため、ここでは記録の有無だけで広めに拾う。
func findCandidateUserIds(db *gorm.DB, fromDate, toDate time.Time) ([]string, error) {
	var userIds []string
	tx := db.Table("records").
		Where("deleted_at IS NULL").
		Where("event_date >= ? AND event_date < ?", fromDate, toDate).
		Distinct("user_id").
		Pluck("user_id", &userIds)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return userIds, nil
}
