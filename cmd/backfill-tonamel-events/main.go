// backfill-tonamel-events は、既存の記録(records)が参照している Tonamel の大会情報を
// tonamel_events テーブルへまとめて投入する初期投入バッチ。
//
// 大会情報は tonamel.com の大会ページをスクレイピングして得るが、一括取得APIが無いため
// 大会ごとに1リクエストかかる。以前はカレンダー表示のたびに参照中の全大会を取り直して
// おり(記録数に比例したN+1)、記録作成時に tonamel_events へ保存する方式へ変更した。
// このバッチは方式変更より前に作られた記録ぶんの大会情報を後から埋めるためのもの。
//
// 冪等性: 既に tonamel_events にあるIDは取得・保存しない(-force で再取得・上書きできる)。
// 何度実行しても重複しない。取得に失敗した大会はスキップしてログに残し、次回実行で再挑戦する。
//
// 使い方:
//
//	# 変更内容を書き込まずに、対象件数の確認のみ行う(デフォルト)
//	go run ./cmd/backfill-tonamel-events
//
//	# 実際に tonamel_events へ保存する
//	go run ./cmd/backfill-tonamel-events -dry-run=false
//
//	# 既に保存済みのものも含めて取り直して上書きする
//	go run ./cmd/backfill-tonamel-events -dry-run=false -force
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

const appName = "backfill-tonamel-events"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// fetchInterval は tonamel.com への取得間隔。外部サイトへ一気に投げないよう1件ごとに待つ。
const fetchInterval = 300 * time.Millisecond

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず対象件数の確認のみ行う")
	force := flag.Bool("force", false, "true の場合、既に保存済みのものも取り直して上書きする")
	// ログは cmd/core-apiserver と同じJSON形式に揃える。これを呼ばないと slog の
	// 既定ハンドラ(テキスト)のままになり、usecase 層のログから layer やソース位置が落ちる。
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

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

	// records が参照している Tonamel大会IDを重複なく集める(論理削除済みは除く)。
	var ids []string
	if tx := db.Table("records").
		Where("tonamel_event_id IS NOT NULL AND tonamel_event_id != '' AND deleted_at IS NULL").
		Distinct().
		Pluck("tonamel_event_id", &ids); tx.Error != nil {
		slog.Error("failed to list tonamel event ids", logging.Err(tx.Error))
		os.Exit(ExitCodeNG)
	}

	slog.Info("found tonamel event ids in records", slog.Int("count", len(ids)))

	// 既に保存済みのIDを除く(-force のときは除かず全件を取り直す)。
	store := infrastructure.NewTonamelEventStore(db)
	targets := ids
	if !*force {
		existing, err := store.FindByIds(ctx, ids)
		if err != nil {
			slog.Error("failed to look up existing tonamel events", logging.Err(err))
			os.Exit(ExitCodeNG)
		}

		existingSet := make(map[string]struct{}, len(existing))
		for _, e := range existing {
			existingSet[e.ID] = struct{}{}
		}

		targets = targets[:0]
		for _, id := range ids {
			if _, ok := existingSet[id]; !ok {
				targets = append(targets, id)
			}
		}
		slog.Info("skipping already saved events",
			slog.Int("targets", len(targets)), slog.Int("already_saved", len(existing)))
	}

	if *dryRun {
		slog.Info("targets to fetch and save",
			slog.Int("targets", len(targets)), slog.Bool("dry_run", true))
		os.Exit(ExitCodeOK)
	}

	fetcher := infrastructure.NewTonamelEvent(slog.Default())

	saved, failed := 0, 0
	for idx, id := range targets {
		if idx > 0 {
			time.Sleep(fetchInterval)
		}

		tonamelEvent, err := fetcher.FindById(ctx, id)
		if err != nil {
			// 取得できない大会(削除済み・非公開など)はスキップ。次回実行で再挑戦できる。
			slog.Warn("failed to fetch tonamel event",
				slog.Int("index", idx+1), slog.Int("total", len(targets)),
				slog.String("tonamel_event_id", id), logging.Err(err))
			failed++
			continue
		}

		if err := store.Save(ctx, tonamelEvent); err != nil {
			slog.Warn("failed to save tonamel event",
				slog.Int("index", idx+1), slog.Int("total", len(targets)),
				slog.String("tonamel_event_id", id), logging.Err(err))
			failed++
			continue
		}

		saved++
	}

	slog.Info("completed", slog.Int("saved", saved), slog.Int("failed", failed))
	os.Exit(ExitCodeOK)
}
