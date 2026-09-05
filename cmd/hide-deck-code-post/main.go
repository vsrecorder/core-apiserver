// hide-deck-code-post は、みんなの公開デッキの投稿(deck_code_posts)を運営が非表示にする、
// または非表示を解除する運用コマンド。API には非表示の口が無く(投稿者にも見せない運営操作の
// ため)、psql で hidden_at を直接書くのは誤った UPDATE の事故につながるので、このコマンドを使う。
//
// 判定基準:
//
//   - 非表示(既定)の対象は「取り下げていない投稿」(unpublished_at IS NULL)で、hidden_at が NULL の
//     ものにだけ現在時刻を入れる。取り下げ済みはもうどこにも出ないので対象にしない
//     (-post-id で指定されても「見つからない」として扱う)。既に非表示なら何もしない。
//   - 解除(-unhide)は hidden_at が入っている投稿を NULL に戻す。取り下げ済みの投稿も対象にする。
//     非表示にした投稿のデッキコードは、取り下げて公開し直そうとしても API が拒否する
//     (usecase の ErrDeckCodePostHidden)ため、解除しないと投稿者はそのコードを二度と公開できない。
//     表示中なら何もしない。
//   - 非表示にした投稿は一覧・個別ページ・いいね・取り込みの対象から外れる。投稿者には
//     「運営により非表示」と表示され、取り下げはできる(adr/deck-code-posts.md 参照)。
//
// 冪等性: 同じ引数で何度実行しても、2回目以降は「skip」になり状態は変わらない。
//
// 使い方:
//
//	# 投稿を1件、非表示にする対象として確認する(既定は dry-run。書き込みなし)
//	go run ./cmd/hide-deck-code-post -post-id <post_id>
//
//	# 実際に非表示にする
//	go run ./cmd/hide-deck-code-post -post-id <post_id> -dry-run=false
//
//	# あるユーザの取り下げていない投稿をまとめて非表示にする
//	go run ./cmd/hide-deck-code-post -user-id <user_id> -dry-run=false
//
//	# 非表示を解除する
//	go run ./cmd/hide-deck-code-post -post-id <post_id> -unhide -dry-run=false
package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// target は操作の候補になる投稿。表示・判定に要る列だけを持つ。
type target struct {
	ID            string
	UserId        string
	DeckName      string
	Code          string
	PublishedAt   time.Time
	UnpublishedAt *time.Time
	HiddenAt      *time.Time
}

// fetchTargets は投稿を -post-id / -user-id の指定で絞って公開日時の新しい順に返す。
// includeWithdrawn が false なら取り下げていない投稿だけ(非表示にする対象)、true なら取り下げ済みも
// 含める(解除の対象)。どちらの指定も空なら何も返さない(全件を対象にする操作は用意しない)。
func fetchTargets(db *gorm.DB, postId string, userId string, includeWithdrawn bool) ([]target, error) {
	if postId == "" && userId == "" {
		return []target{}, nil
	}

	q := db.Table("deck_code_posts AS p").
		Select("p.id AS id, p.user_id AS user_id, d.name AS deck_name, c.code AS code, p.published_at AS published_at, p.unpublished_at AS unpublished_at, p.hidden_at AS hidden_at").
		Joins("JOIN decks d ON d.id = p.deck_id").
		Joins("JOIN deck_codes c ON c.id = p.deck_code_id")
	if !includeWithdrawn {
		q = q.Where("p.unpublished_at IS NULL")
	}
	if postId != "" {
		q = q.Where("p.id = ?", postId)
	}
	if userId != "" {
		q = q.Where("p.user_id = ?", userId)
	}

	var targets []target
	if tx := q.Order("p.published_at DESC").Scan(&targets); tx.Error != nil {
		return nil, tx.Error
	}
	if targets == nil {
		targets = []target{}
	}

	return targets, nil
}

// planActions は候補を「今回変更するもの」と「既にその状態でスキップするもの」に分ける。
// 非表示なら hidden_at が空のものだけ、解除なら hidden_at が入っているものだけを変更する。
func planActions(targets []target, unhide bool) (apply []target, skip []target) {
	apply = []target{}
	skip = []target{}
	for _, t := range targets {
		hidden := t.HiddenAt != nil
		if hidden == unhide {
			apply = append(apply, t)
		} else {
			skip = append(skip, t)
		}
	}

	return apply, skip
}

// hide は対象を非表示にする。条件を再確認して(取り下げ済み・既に非表示は除く)更新した件数を返す。
func hide(db *gorm.DB, ids []string, at time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx := db.Exec(
		"UPDATE deck_code_posts SET hidden_at = ?, updated_at = ? WHERE id IN ? AND unpublished_at IS NULL AND hidden_at IS NULL",
		at, at, ids,
	)

	return tx.RowsAffected, tx.Error
}

// unhide は非表示を解除する。取り下げ済みの投稿も対象で、表示中のものは更新しない。
func unhide(db *gorm.DB, ids []string, at time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx := db.Exec(
		"UPDATE deck_code_posts SET hidden_at = NULL, updated_at = ? WHERE id IN ? AND hidden_at IS NOT NULL",
		at, ids,
	)

	return tx.RowsAffected, tx.Error
}

func describe(t target) string {
	state := "visible"
	if t.HiddenAt != nil {
		state = "hidden since " + t.HiddenAt.Format(time.RFC3339)
	}
	if t.UnpublishedAt != nil {
		state += ", withdrawn " + t.UnpublishedAt.Format(time.RFC3339)
	}

	return t.ID + " user=" + t.UserId + " deck=" + t.DeckName + " code=" + t.Code +
		" published=" + t.PublishedAt.Format(time.RFC3339) + " (" + state + ")"
}

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず対象の確認のみ行う")
	postId := flag.String("post-id", "", "対象の投稿ID(deck_code_posts.id)")
	userId := flag.String("user-id", "", "対象を絞るユーザID。指定した場合、そのユーザの投稿がまとめて対象になる")
	doUnhide := flag.Bool("unhide", false, "true の場合、非表示を解除する(取り下げ済みの投稿も対象)")
	flag.Parse()

	if *postId == "" && *userId == "" {
		log.Printf("either -post-id or -user-id is required\n")
		flag.Usage()
		os.Exit(ExitCodeNG)
	}

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

	// 解除は取り下げ済みの投稿も対象にする(非表示のまま取り下げたコードの公開し直しを許すため)
	targets, err := fetchTargets(db, *postId, *userId, *doUnhide)
	if err != nil {
		log.Printf("failed to query deck code posts: %v\n", err)
		os.Exit(ExitCodeNG)
	}
	if len(targets) == 0 {
		log.Printf("no deck code post matched post-id=%q user-id=%q (withdrawn posts are excluded unless -unhide)\n", *postId, *userId)
		os.Exit(ExitCodeNG)
	}

	action, done := "hide", "hidden"
	if *doUnhide {
		action, done = "unhide", "unhidden"
	}

	apply, skip := planActions(targets, *doUnhide)
	for _, t := range skip {
		log.Printf("skip (already in the requested state): %s\n", describe(t))
	}
	for _, t := range apply {
		if *dryRun {
			log.Printf("[dry-run] would %s: %s\n", action, describe(t))
		} else {
			log.Printf("%s: %s\n", action, describe(t))
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d posts would be %s, %d skipped. no changes were made\n", len(apply), done, len(skip))
		os.Exit(ExitCodeOK)
	}

	ids := make([]string, 0, len(apply))
	for _, t := range apply {
		ids = append(ids, t.ID)
	}

	now := time.Now().Local()
	var affected int64
	if *doUnhide {
		affected, err = unhide(db, ids, now)
	} else {
		affected, err = hide(db, ids, now)
	}
	if err != nil {
		log.Printf("failed to %s deck code posts: %v\n", action, err)
		os.Exit(ExitCodeNG)
	}

	log.Printf("completed: %d posts %s (%d planned, %d skipped)\n", affected, done, len(apply), len(skip))
	os.Exit(ExitCodeOK)
}
