/*
 * push が届いているかを platform 別に点検する。
 *
 * push の失敗は静かに起きる。配信バッチは送出に失敗しても通知は残す設計なので
 * バッチのログは正常に見え、アプリ内通知も普通に増える。さらにプッシュサービスごとに
 * VAPID の検証の厳しさが違い、Apple は JWT を厳密に見るのに対して FCM のレガシー endpoint は
 * 署名を検証せず 201 を返す。そのため設定を1つ間違えると「Android は成功しているのに
 * iOS だけ一通も届かない」という壊れ方をする。
 *
 * 実際に VAPID の subject が二重の mailto: になっていた時期、iOS への配信が11日間
 * 全滅していたのに気付けなかった。全体の成功率では Android の成功に薄まって見えるため、
 * platform 別に「成功が1件も無い」を拾う必要がある。
 *
 * 使い方:
 *
 *	# 直近7日の配達を点検する(既定)
 *	go run ./cmd/check-push-health
 *
 *	# 異常があったときだけ Slack へ通知する(定期実行用。SLACK_WEBHOOK_URL が必要)
 *	./bin/check-push-health -notify-slack
 *
 *	# 期間を変える
 *	./bin/check-push-health -days=1
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
	"github.com/vsrecorder/core-apiserver/internal/slack"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "check-push-health"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	days := flag.Int("days", 7, "何日ぶんの配達を点検するか")
	notifySlack := flag.Bool("notify-slack", false, "true の場合、異常が見つかったら SLACK_WEBHOOK_URL へ通知する")
	flag.Parse()

	if *days < 1 {
		slog.Error("-days must be 1 or greater", slog.Int("days", *days))
		os.Exit(ExitCodeNG)
	}

	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", logging.Err(err))
	}

	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	// 通知するつもりで通知できない状態は、黙って点検だけして終わるより早く気付きたい
	if *notifySlack && slackWebhookURL == "" {
		slog.Error("SLACK_WEBHOOK_URL is not set: -notify-slack requires it")
		os.Exit(ExitCodeNG)
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
	since := time.Now().AddDate(0, 0, -*days)

	stats, problems, err := usecase.NewPushHealth(infrastructure.NewPushDelivery(db)).Check(ctx, since)
	if err != nil {
		slog.Error("failed to check push health", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	// 異常の有無によらず内訳は残す。後から「いつから落ちていたか」を追えるようにする
	for _, stat := range stats {
		slog.Info("push delivery stat",
			slog.String("platform", stat.Platform),
			slog.Int("total", stat.Total),
			slog.Int("sent", stat.Sent),
			slog.String("success_rate", formatRate(stat.SuccessRate())),
			slog.Int("top_failure_status_code", stat.TopFailureStatusCode),
		)
		fmt.Printf(
			"%-8s %4d/%-4d 成功 (%s) 最多の失敗=%d\n",
			stat.Platform, stat.Sent, stat.Total, formatRate(stat.SuccessRate()), stat.TopFailureStatusCode,
		)
	}

	if len(stats) == 0 {
		slog.Info("no delivery in the period", slog.Int("days", *days))
	}

	for _, problem := range problems {
		// critical(1件も届いていない)は設定の事故である可能性が高いので ERROR で出す
		level := slog.LevelWarn
		if problem.Severity == usecase.PushHealthSeverityCritical {
			level = slog.LevelError
		}

		slog.Log(ctx, level, "push delivery is failing",
			slog.String("platform", problem.Stat.Platform),
			slog.String("severity", string(problem.Severity)),
			slog.Int("total", problem.Stat.Total),
			slog.Int("sent", problem.Stat.Sent),
			slog.Int("top_failure_status_code", problem.Stat.TopFailureStatusCode),
		)
	}

	// 通知は点検結果を出し切ってから行う。Slack が落ちていてもログには残るようにする
	if *notifySlack && len(problems) > 0 {
		if err := slack.Notify(slackWebhookURL, buildSlackMessage(*days, problems)); err != nil {
			slog.Error("failed to notify to slack", logging.Err(err))
			os.Exit(ExitCodeNG)
		}

		slog.Info("notified the problems to slack", slog.Int("problems", len(problems)))
	}

	// 異常を見つけたこと自体は「バッチの失敗」ではないので OK で終える。
	// cron の実行失敗と点検結果を混ぜると、どちらも見なくなる
	os.Exit(ExitCodeOK)
}

func formatRate(rate float64) string {
	return fmt.Sprintf("%.0f%%", rate*100)
}

// hintOf は失敗のステータスコードから、次に見るべき場所を示す。
// 通知を見た人がすぐ動けるようにするためで、判定そのものには使わない。
func hintOf(statusCode int) string {
	switch statusCode {
	case 403:
		return "VAPID の設定(subject・鍵ペア)を疑う。`./bin/verify-vapid-keys` で確認できる"
	case 404, 410:
		return "購読切れ。端末側で購読が作り直されるまで届かない"
	case 400, 413:
		return "送信内容(ペイロード・ヘッダ)を疑う"
	case 429:
		return "プッシュサービスからの絞り込み。送信頻度を疑う"
	default:
		return "配信バッチのログで response_body を確認する"
	}
}

// buildSlackMessage は異常の内容を Slack へ投稿する本文へ組み立てる。
func buildSlackMessage(days int, problems []*usecase.PushHealthProblem) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(":rotating_light: push が届いていません(直近%d日)\n\n", days))

	for _, problem := range problems {
		stat := problem.Stat

		sb.WriteString(fmt.Sprintf(
			"• *%s* %d/%d 件成功 (%s)",
			stat.Platform, stat.Sent, stat.Total, formatRate(stat.SuccessRate()),
		))

		if problem.Severity == usecase.PushHealthSeverityCritical {
			sb.WriteString(" ← 1件も届いていない")
		}

		sb.WriteString("\n")

		if stat.TopFailureStatusCode > 0 {
			sb.WriteString(fmt.Sprintf(
				"    最多の失敗: %d — %s\n",
				stat.TopFailureStatusCode, hintOf(stat.TopFailureStatusCode),
			))
		}
	}

	sb.WriteString("\nプッシュサービスによって VAPID の検証の厳しさが違うため、")
	sb.WriteString("片方の platform だけ落ちることがあります。")

	return sb.String()
}
