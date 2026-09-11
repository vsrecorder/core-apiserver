// auto-like-deck-code-posts は、みんなの公開デッキへ投稿されたデッキに運営の公式アカウントで
// いいねを付ける定期バッチ(config/crontab から10分ごとに実行)。
//
// 公開しても反応が無いと投稿は続かないため、運営が最初の1つを必ず付ける。
//
// 対象の判定基準:
//
//   - 閲覧者向けに公開中(取り下げ済み・運営の非表示でない)の投稿。
//   - 公式アカウントがまだいいねしていない投稿(既に押していれば対象にならない)。
//     いいねを外した投稿は未いいねに戻るため、次の実行でまた付く点に注意する。
//   - 公式アカウント自身の投稿は対象にしない(自作自演になるため)。
//   - -since を指定した場合は、その日の0時以降に公開された投稿だけ。未指定なら公開中の全件。
//
// 冪等性: 未いいねの投稿だけを対象にし、書き込みも ON CONFLICT DO NOTHING なので、多重起動・
// 再実行しても同じ投稿に二重のいいねは付かない。途中で失敗しても次の実行が残りを拾う。
//
// 付けたいいねは投稿者への日次まとめ通知(cmd/notify-deck-code-post-likes)には出さない。
// 自動で全投稿に付くため、通知すると毎日同じ文面が投稿者全員へ届き、人が押したいいねの
// 知らせが埋もれるため。あちらも同じ OFFICIAL_USER_ID を読んで除外する。
//
// いいねを押す公式アカウントのユーザIDは .env の OFFICIAL_USER_ID から読む(未設定なら何もせず終了)。
//
//	go run ./cmd/auto-like-deck-code-posts                                   # 対象を確認するだけ(dry-run)
//	go run ./cmd/auto-like-deck-code-posts -dry-run=false                    # いいねを付ける
//	go run ./cmd/auto-like-deck-code-posts -since=2026-09-12 -dry-run=false  # その日以降に公開された投稿だけ
//	go run ./cmd/auto-like-deck-code-posts -user-id=xxxx -dry-run=false      # 特定の投稿者の投稿だけ
//	go run ./cmd/auto-like-deck-code-posts -limit=50 -dry-run=false          # 1回に付ける上限を変える
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
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "auto-like-deck-code-posts"

// defaultLimit は1回の実行で押すいいねの上限。10分ごとの実行でこれを超える新規投稿は
// 現状ありえないため、超えたときは設定ミスか想定外の事態とみなして次の実行に回す
// (初回に過去分をまとめて付けるときは -limit で引き上げる)。
const defaultLimit = 200

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	// ログは cmd/core-apiserver と同じJSON形式に揃える(理由は AGENTS.md「ログ」参照)。
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	dryRun := flag.Bool("dry-run", true, "true の場合、いいねは付けず対象の件数を確認するだけにする")
	userId := flag.String("user-id", "", "指定した場合、その投稿者の投稿だけを対象にする")
	sinceFlag := flag.String("since", "", "対象にする公開日の下限(YYYY-MM-DD)。未指定なら公開中の全件")
	limit := flag.Int("limit", defaultLimit, "1回の実行で付けるいいねの上限")
	flag.Parse()

	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", logging.Err(err))
	}

	likerUserId := os.Getenv("OFFICIAL_USER_ID")
	if likerUserId == "" {
		slog.Error("OFFICIAL_USER_ID is not set")
		os.Exit(ExitCodeNG)
	}

	var publishedFrom time.Time
	if *sinceFlag != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, *sinceFlag, time.Local)
		if err != nil {
			slog.Error("invalid -since", slog.String("since", *sinceFlag), logging.Err(err))
			os.Exit(ExitCodeNG)
		}
		publishedFrom = parsed
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

	autoLiker := usecase.NewDeckCodePostAutoLiker(
		infrastructure.NewDeckCodePost(db),
		likerUserId,
	)

	// dry-run と対象は属性で出す(メッセージを分岐させると grep の条件が増える)。
	// target_user_id / since が空文字なら絞り込み無し。
	batchAttrs := []any{
		slog.String("liker_user_id", likerUserId),
		slog.String("target_user_id", *userId),
		slog.String("since", *sinceFlag),
		slog.Int("limit", *limit),
		slog.Bool("dry_run", *dryRun),
	}
	slog.Info("liking deck code posts", batchAttrs...)

	count, err := autoLiker.LikeUnliked(context.Background(), *userId, publishedFrom, *limit, *dryRun)
	if err != nil {
		slog.Error("failed to like deck code posts",
			append(batchAttrs, slog.Int("liked_posts", count), logging.Err(err))...)
		os.Exit(ExitCodeNG)
	}

	slog.Info("completed", append(batchAttrs, slog.Int("liked_posts", count))...)

	os.Exit(ExitCodeOK)
}
