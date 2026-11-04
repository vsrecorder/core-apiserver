// notify-period-report は、月や環境が切り替わった日に、終わった期間のバトルレポートを
// ユーザーへ届ける定期バッチ。
//
//   - 月の切り替わり(毎月1日): 先月に1戦以上記録した人へ「10月のバトルレポートができました」
//     (/users/report/{YYYY-MM})
//   - 環境の切り替わり(新しい環境の初日 = 前の環境の to_date の翌日): 終わった環境で1戦以上
//     記録した人へ「『○○』環境のバトルレポートができました」(/users/report/environments/{id})
//
// どちらも週次バトルレポート(notify-weekly-report)と同じ「記録の配当」で、購読の有無によらず
// アプリ内通知を作り、push 購読者には push も送る。0戦の人には何も送らない。
// 月初と環境の切り替わりが同じ日に重なれば、両方を送る。
//
// 毎日起動し、その日が月初でも環境の切り替わりでもなければ何もせずに終わる。
// 判定は実行日(-date)だけで決まるため、取りこぼした日の分は -date か -month / -environment-id を
// 指定して手で流す。同じ月・同じ環境の通知は二重に作らないので、再実行しても安全。
//
// 想定運用: OSのcronから毎日 12:00 JST に起動する(crontab例は config/crontab 参照)。
// 既存の通知(月 08:00・金/日 20:00・毎日 21:00)と時刻が重ならないように昼にしている。
//
// 使い方:
//
//	# 送信せず対象者と件数だけ確認する(デフォルト。判定は今日)
//	go run ./cmd/notify-period-report
//
//	# 実際に通知を作成する
//	go run ./cmd/notify-period-report -dry-run=false
//
//	# 実行日を指定して判定する(取りこぼしの再送や検証用。2026-11-27 なら m6a が終わった日)
//	go run ./cmd/notify-period-report -date=2026-11-27
//
//	# 対象を直接指定する(-date の判定は使わない)
//	go run ./cmd/notify-period-report -month=2026-10
//	go run ./cmd/notify-period-report -environment-id=m6a
//
//	# 特定ユーザーのみ対象にする(検証用)
//	go run ./cmd/notify-period-report -month=2026-10 -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "notify-period-report"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

const (
	dateLayout      = "2006-01-02"
	yearMonthLayout = "2006-01"
)

// target は1回の実行で送る通知1種類ぶん(ある月、またはある環境)。
type target struct {
	kind string // "month" / "environment"
	key  string // YYYY-MM / 環境ID
	from time.Time
	to   time.Time
}

func main() {
	// ログは cmd/core-apiserver と同じJSON形式に揃える(notify-weekly-report と同じ理由)
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	dryRun := flag.Bool("dry-run", true, "true の場合、通知は作成せず対象者の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全対象ユーザー)")
	dateFlag := flag.String("date", "", "実行日(YYYY-MM-DD)。この日が月初・環境の初日かで対象を決める。未指定なら今日")
	monthFlag := flag.String("month", "", "対象月(YYYY-MM)を直接指定する。指定すると -date の判定は使わない")
	environmentFlag := flag.String("environment-id", "", "対象環境のIDを直接指定する。指定すると -date の判定は使わない")
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

	ctx := context.Background()
	environmentRepo := infrastructure.NewEnvironment(db)

	targets, err := resolveTargets(ctx, environmentRepo, *dateFlag, *monthFlag, *environmentFlag, time.Now())
	if err != nil {
		slog.Error("failed to resolve targets", logging.Err(err))
		os.Exit(ExitCodeNG)
	}
	if len(targets) == 0 {
		slog.Info("nothing to send: not the first day of a month or an environment", slog.String("date", *dateFlag))
		os.Exit(ExitCodeOK)
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

	officialEventEnvironmentRepo := infrastructure.NewOfficialEventEnvironment(db)
	notifier := usecase.NewPeriodReportNotifier(
		infrastructure.NewUserStat(db),
		infrastructure.NewDeckUsageStat(db),
		infrastructure.NewNotification(db),
		pushNotifier,
		environmentRepo,
		officialEventEnvironmentRepo,
	)

	failed := false
	for _, t := range targets {
		var userIds []string
		if *targetUserId != "" {
			userIds = []string{*targetUserId}
		} else {
			userIds, err = findCandidateUserIds(db, t)
			if err != nil {
				slog.Error("failed to list candidate users", slog.String("kind", t.kind), slog.String("key", t.key), logging.Err(err))
				failed = true
				continue
			}
		}

		// dry-run はメッセージではなく属性で出す(分岐させると grep の条件が増える)
		batchAttrs := []any{
			slog.String("kind", t.kind),
			slog.String("key", t.key),
			slog.Int("candidate_users", len(userIds)),
			slog.Bool("dry_run", *dryRun),
		}
		slog.Info("sending period-report", batchAttrs...)

		sent := 0
		for _, userId := range userIds {
			var ok bool
			if t.kind == "month" {
				ok, err = notifier.NotifyMonthly(ctx, userId, t.key, *dryRun)
			} else {
				ok, err = notifier.NotifyEnvironment(ctx, userId, t.key, *dryRun)
			}
			if err != nil {
				slog.Error("failed to notify user",
					slog.String("user_id", userId), slog.String("kind", t.kind), slog.String("key", t.key), logging.Err(err))
				continue
			}
			if ok {
				sent++
				slog.Info("notified user",
					slog.String("user_id", userId), slog.String("kind", t.kind), slog.String("key", t.key), slog.Bool("dry_run", *dryRun))
			}
		}

		slog.Info("completed", append(batchAttrs, slog.Int("notified_users", sent))...)
	}

	if failed {
		os.Exit(ExitCodeNG)
	}
	os.Exit(ExitCodeOK)
}

// environmentFinder は resolveTargets が使う環境の取得(テストで差し替えるため最小限にしている)。
type environmentFinder interface {
	Find(ctx context.Context) ([]*entity.Environment, error)
	FindById(ctx context.Context, id string) (*entity.Environment, error)
}

// resolveTargets はこの実行で送る月・環境を決める。
//   - -month / -environment-id の指定があればそれだけを返す
//   - 無ければ実行日(-date、未指定なら now)で判定する。1日なら先月、前日に終わった(to_date が
//     前日の)環境があればその環境
func resolveTargets(
	ctx context.Context,
	environmentRepo environmentFinder,
	date string,
	month string,
	environmentId string,
	now time.Time,
) ([]target, error) {
	if month != "" || environmentId != "" {
		var targets []target
		if month != "" {
			t, err := time.ParseInLocation(yearMonthLayout, month, time.Local)
			if err != nil {
				return nil, fmt.Errorf("invalid -month %q: %w", month, err)
			}
			targets = append(targets, monthTarget(t))
		}
		if environmentId != "" {
			env, err := environmentRepo.FindById(ctx, environmentId)
			if err != nil {
				return nil, fmt.Errorf("environment %q: %w", environmentId, err)
			}
			targets = append(targets, environmentTarget(env))
		}
		return targets, nil
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if date != "" {
		t, err := time.ParseInLocation(dateLayout, date, time.Local)
		if err != nil {
			return nil, fmt.Errorf("invalid -date %q: %w", date, err)
		}
		today = t
	}

	var targets []target
	if today.Day() == 1 {
		targets = append(targets, monthTarget(today.AddDate(0, -1, 0)))
	}

	envs, err := environmentRepo.Find(ctx)
	if err != nil {
		return nil, err
	}
	yesterday := today.AddDate(0, 0, -1)
	for _, env := range envs {
		// GORM の DATE は UTC 0時で返るため、時刻ではなく暦日で比べる
		y, m, d := env.ToDate.Date()
		if y == yesterday.Year() && m == yesterday.Month() && d == yesterday.Day() {
			targets = append(targets, environmentTarget(env))
		}
	}

	return targets, nil
}

func monthTarget(t time.Time) target {
	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	return target{kind: "month", key: from.Format(yearMonthLayout), from: from, to: from.AddDate(0, 1, 0)}
}

func environmentTarget(env *entity.Environment) target {
	fy, fm, fd := env.FromDate.Date()
	ty, tm, td := env.ToDate.Date()
	return target{
		kind: "environment",
		key:  env.ID,
		from: time.Date(fy, fm, fd, 0, 0, 0, 0, time.Local),
		// to_date は含む日付なので翌日0時を exclusive 上限にする
		to: time.Date(ty, tm, td, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1),
	}
}

// findCandidateUserIds は「期間内に開催日(event_date)を持つ記録があるユーザー」を候補として返す。
// 環境は、開催日が期間外でもこの環境として数える公式イベント(official_event_environments)の
// 記録を持つユーザーも候補に入れる。「1戦以上あるか」「集計対象外を除いて残るか」の
// 最終判定は usecase 側が戦績集計で行う(ここは広めに拾うだけ)。
func findCandidateUserIds(db *gorm.DB, t target) ([]string, error) {
	query := db.Table("records").
		Where("deleted_at IS NULL")

	if t.kind == "environment" {
		query = query.Where(
			"(event_date >= ? AND event_date < ?) OR official_event_id IN (SELECT official_event_id FROM official_event_environments WHERE environment_id = ?)",
			t.from, t.to, t.key,
		)
	} else {
		query = query.Where("event_date >= ? AND event_date < ?", t.from, t.to)
	}

	var userIds []string
	if tx := query.Distinct("user_id").Pluck("user_id", &userIds); tx.Error != nil {
		return nil, tx.Error
	}
	return userIds, nil
}
