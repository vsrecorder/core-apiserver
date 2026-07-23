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
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// fetchInterval は tonamel.com への取得間隔。外部サイトへ一気に投げないよう1件ごとに待つ。
const fetchInterval = 300 * time.Millisecond

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず対象件数の確認のみ行う")
	force := flag.Bool("force", false, "true の場合、既に保存済みのものも取り直して上書きする")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

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

	ctx := context.Background()

	// records が参照している Tonamel大会IDを重複なく集める(論理削除済みは除く)。
	var ids []string
	if tx := db.Table("records").
		Where("tonamel_event_id IS NOT NULL AND tonamel_event_id != '' AND deleted_at IS NULL").
		Distinct().
		Pluck("tonamel_event_id", &ids); tx.Error != nil {
		log.Printf("failed to list tonamel event ids: %v\n", tx.Error)
		os.Exit(ExitCodeNG)
	}

	log.Printf("records から %d 件のTonamel大会IDを検出\n", len(ids))

	// 既に保存済みのIDを除く(-force のときは除かず全件を取り直す)。
	store := infrastructure.NewTonamelEventStore(db)
	targets := ids
	if !*force {
		existing, err := store.FindByIds(ctx, ids)
		if err != nil {
			log.Printf("failed to look up existing tonamel events: %v\n", err)
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
		log.Printf("うち未保存は %d 件(保存済み %d 件はスキップ)\n", len(targets), len(existing))
	}

	if *dryRun {
		log.Printf("[dry-run] %d 件を取得・保存対象とします(-dry-run=false で実行)\n", len(targets))
		os.Exit(ExitCodeOK)
	}

	fetcher := infrastructure.NewTonamelEvent(logger)

	saved, failed := 0, 0
	for idx, id := range targets {
		if idx > 0 {
			time.Sleep(fetchInterval)
		}

		tonamelEvent, err := fetcher.FindById(ctx, id)
		if err != nil {
			// 取得できない大会(削除済み・非公開など)はスキップ。次回実行で再挑戦できる。
			log.Printf("[%d/%d] 取得失敗 id=%s: %v\n", idx+1, len(targets), id, err)
			failed++
			continue
		}

		if err := store.Save(ctx, tonamelEvent); err != nil {
			log.Printf("[%d/%d] 保存失敗 id=%s: %v\n", idx+1, len(targets), id, err)
			failed++
			continue
		}

		saved++
	}

	log.Printf("完了: 保存 %d 件 / 失敗 %d 件\n", saved, failed)
	os.Exit(ExitCodeOK)
}
