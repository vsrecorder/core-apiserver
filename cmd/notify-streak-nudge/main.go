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
// B-1(Web Push)導入後は、アプリ内通知を作った上で購読端末へ push も送る(配達手段の追加であり
// 判定は変えない)。.env の VAPID_* が未設定なら push は送らずアプリ内通知だけになる。
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
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "notify-streak-nudge"

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

	dryRun := flag.Bool("dry-run", true, "true の場合、通知は作成せず対象者の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全対象ユーザー)")
	flag.Parse()

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
		slog.Warn("web push is disabled: VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY / VAPID_SUBJECT are not set. in-app notifications only")
	}
	pushNotifier := usecase.NewPushNotifier(
		infrastructure.NewPushSubscription(db),
		infrastructure.NewPushDelivery(db),
		pushSender,
	)

	streakNudge := usecase.NewStreakNudge(
		infrastructure.NewUserStreak(db),
		infrastructure.NewNotification(db),
		pushNotifier,
	)

	ctx := context.Background()

	var userIds []string
	if *targetUserId != "" {
		userIds = []string{*targetUserId}
	} else {
		userIds, err = findCandidateUserIds(db)
		if err != nil {
			slog.Error("failed to list candidate users", logging.Err(err))
			os.Exit(ExitCodeNG)
		}
	}

	// dry-run はメッセージではなく属性で出す(分岐させると grep の条件が増える)
	batchAttrs := []any{
		slog.Int("candidate_users", len(userIds)),
		slog.Bool("dry_run", *dryRun),
	}
	slog.Info("sending streak-nudge", batchAttrs...)

	sent := 0
	for _, userId := range userIds {
		ok, err := streakNudge.NudgeUser(ctx, userId, *dryRun)
		if err != nil {
			slog.Error("failed to nudge user", slog.String("user_id", userId), logging.Err(err))
			continue
		}
		if ok {
			sent++
			slog.Info("nudged user", slog.String("user_id", userId), slog.Bool("dry_run", *dryRun))
		}
	}

	slog.Info("completed", append(batchAttrs, slog.Int("nudged_users", sent))...)

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
