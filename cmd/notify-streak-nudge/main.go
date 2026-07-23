// notify-streak-nudge は、今週まだ記録がなく、今週記録しないと連続記録(ストリーク)が
// 途切れてしまう瀬戸際のユーザーを抽出し、途切れ防止のアプリ内通知を作成する定期バッチ。
//
// 「2週連続の壁」で離脱しやすい層(1週・2〜3週ストリーク層)を、その週のうちに記録へ
// 押し戻すことを狙う(growth-plan-2026Q3.md 施策B-5 / B5_STREAK_NUDGE_PLAN.md)。
//
// 瀬戸際かどうかの判定は usecase.NudgeUser が updateStreak/isStreakExpired と同じ
// 週・フリーズ猶予の基準で行うため、フリーズでまだ救える余裕のある人は対象にしない。
// 同一週の二重送信はガードされるため、cronの多重起動でも安全。
//
// 想定運用: OSのcronから毎週日曜夜に起動する(crontab例は B5_STREAK_NUDGE_PLAN.md 参照)。
//
// 使い方:
//
//	# 送信せず対象者と件数だけ確認する(デフォルト)
//	go run ./cmd/notify-streak-nudge
//
//	# 実際に通知を作成する
//	go run ./cmd/notify-streak-nudge -dry-run=false
//
//	# 特定ユーザーのみ対象にする(検証用)
//	go run ./cmd/notify-streak-nudge -user-id=xxxxx -dry-run=false
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

	streakNudge := usecase.NewStreakNudge(
		infrastructure.NewUserStreak(db),
		infrastructure.NewNotification(db),
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
		log.Printf("[dry-run] checking streak-nudge targets among %d users (通知は作成しません)\n", len(userIds))
	} else {
		log.Printf("sending streak-nudge among %d users\n", len(userIds))
	}

	sent := 0
	for _, userId := range userIds {
		ok, err := streakNudge.NudgeUser(ctx, userId, *dryRun)
		if err != nil {
			log.Printf("failed to nudge user=%s: %v\n", userId, err)
			continue
		}
		if ok {
			sent++
			if *dryRun {
				log.Printf("[dry-run] TARGET user=%s\n", userId)
			} else {
				log.Printf("nudged user=%s\n", userId)
			}
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d users are streak-nudge targets\n", sent, len(userIds))
	} else {
		log.Printf("completed: nudged %d/%d users\n", sent, len(userIds))
	}

	os.Exit(ExitCodeOK)
}

// findCandidateUserIds は連続記録中(current_weeks >= 1)のユーザーを候補として返す。
// current_weeks が 0 のユーザーは守るべき連続が無い(または既に途切れている)ため対象外。
// 「今週が瀬戸際か」の最終判定は usecase.NudgeUser が current_weeks ではなく
// last_recorded_week とフリーズ枠から行う(current_weeks は参照時点で古い可能性があるため)。
func findCandidateUserIds(db *gorm.DB) ([]string, error) {
	var userIds []string
	tx := db.Table("user_streaks").
		Where("current_weeks >= ?", 1).
		Distinct("user_id").
		Pluck("user_id", &userIds)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return userIds, nil
}
