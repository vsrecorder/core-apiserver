// backfill-deck-code-post-acespec は、みんなの公開デッキの投稿のうち ACE SPEC が空のものを
// deckcard-api で判定し直して埋める運用バッチ。
//
// 公開時にも同じ判定を行うが、deckcard-api が応答しなかった場合は公開を止めず
// 「判定なし」(空)のまま保存する(internal/usecase/deck_code_post.go の findAceSpec)。
// 空のまま残ると、画面はその都度 acespec API を引いて補えるが、OGP 画像と ACE SPEC での
// 絞り込みは保存値を使うため反映されない。このバッチはその取りこぼしを後から埋める。
//
// 判定基準:
//
//   - 対象は「取り下げていない投稿」(unpublished_at IS NULL)で ace_spec_card_id が空のもの。
//     取り下げ済みの投稿はもうどこにも出ないため対象にしない。
//   - deckcard-api が ACE SPEC を返したらカードID・カード名・画像URLを保存する。
//   - 「そのデッキに ACE SPEC が入っていない」(204)場合は空が正しい状態なので、何も書かない。
//     この投稿は次回以降も対象に挙がるが、書き込みは発生しない。
//   - 問い合わせに失敗した投稿はログに残して次へ進む(1件の失敗で全体を止めない)。
//
// OGP 画像:
//
//   - 保存値を埋めると OGP 画像の内容(ACE SPEC の帯)が変わる。画像は内容の指紋をキーにして
//     CDN に置かれるため、埋めた後の投稿は「別のキーの画像」になり、まだ存在しない状態になる。
//   - そこで書き込んだ投稿については個別ページ(webapp)を1回取得し、新しい画像を作らせる
//     (webapp は generateMetadata で画像が無ければ生成してアップロードする)。
//     -refresh-ogp=false で省略できる。接続先は WEBAPP_BASE_URL(未設定なら公開ドメイン)。
//
// 冪等性: 書き込むのは「空の投稿に判定結果が得られたとき」だけなので、続けて実行しても
// 2回目以降は何も変わらない(ACE SPEC が入っていないデッキだけ毎回問い合わせが走る)。
//
// 使い方:
//
//	# 対象と判定結果を確認するだけ(既定。書き込みも画像生成もしない)
//	go run ./cmd/backfill-deck-code-post-acespec
//
//	# 実際に埋める
//	go run ./cmd/backfill-deck-code-post-acespec -dry-run=false
//
//	# 特定のユーザ・投稿だけ
//	go run ./cmd/backfill-deck-code-post-acespec -user-id <user_id> -dry-run=false
//	go run ./cmd/backfill-deck-code-post-acespec -post-id <post_id> -dry-run=false
//
//	# 件数を絞る / OGP 画像の作り直しを省く
//	go run ./cmd/backfill-deck-code-post-acespec -limit 20 -refresh-ogp=false -dry-run=false
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// webappDefaultBaseURL は個別ページの取得先の既定。DECKCARD_API_BASE_URL と同じく、
// 専用の環境変数(WEBAPP_BASE_URL)が無ければ公開ドメインを使う。
const webappDefaultBaseURL = "https://vsrecorder.mobi"

// ogpRefreshTimeout は個別ページの取得(＝OGP 画像の生成)を待つ上限。
// 画像の生成は satori の描画とアップロードを伴い数百 ms〜数秒かかる。
const ogpRefreshTimeout = 30 * time.Second

// requestInterval は投稿ごとの間隔。deckcard-api と webapp に短時間で連続して
// 投げないよう、1件ごとに少し空ける。
const requestInterval = 300 * time.Millisecond

// target は判定し直す候補の投稿。
type target struct {
	ID       string
	UserId   string
	DeckName string
	Code     string
}

type options struct {
	dryRun bool
	// userId / postId は対象を絞る。空なら絞らない。
	userId string
	postId string
	// limit は1回の実行で扱う最大件数。0 なら制限しない。
	limit int
	// refreshOgp が true なら、書き込んだ投稿の個別ページを取得して OGP 画像を作り直す。
	refreshOgp bool
}

type result struct {
	// updated は保存値を埋めた投稿の数(dry-run では「埋める予定」の数)。
	updated int
	// noAceSpec は deckcard-api が「入っていない」と答えた投稿の数。
	noAceSpec int
	// failed は問い合わせ・保存・画像生成に失敗した投稿の数。
	failed int
}

// webappBaseURLFromEnv は個別ページの取得先(未設定なら公開ドメイン)を返す。
// godotenv.Load() の後に呼ぶ必要があるため関数にしている。
func webappBaseURLFromEnv() string {
	if v := strings.TrimRight(os.Getenv("WEBAPP_BASE_URL"), "/"); v != "" {
		return v
	}

	return webappDefaultBaseURL
}

// deckCodePostPageURL は個別ページの URL。パスは webapp のルーティングと揃える
// (webapp: src/app/shared_decks/[id]/page.tsx)。
func deckCodePostPageURL(baseURL string, postId string) string {
	return strings.TrimRight(baseURL, "/") + "/shared_decks/" + postId
}

// fetchTargets は ACE SPEC が空の「取り下げていない投稿」を公開日時の新しい順に返す。
func fetchTargets(db *gorm.DB, opts options) ([]target, error) {
	q := db.Table("deck_code_posts AS p").
		Select("p.id AS id, p.user_id AS user_id, d.name AS deck_name, c.code AS code").
		Joins("JOIN decks d ON d.id = p.deck_id").
		Joins("JOIN deck_codes c ON c.id = p.deck_code_id").
		Where("p.unpublished_at IS NULL").
		Where("p.ace_spec_card_id = ''")
	if opts.userId != "" {
		q = q.Where("p.user_id = ?", opts.userId)
	}
	if opts.postId != "" {
		q = q.Where("p.id = ?", opts.postId)
	}
	q = q.Order("p.published_at DESC")
	if opts.limit > 0 {
		q = q.Limit(opts.limit)
	}

	var targets []target
	if tx := q.Scan(&targets); tx.Error != nil {
		return nil, tx.Error
	}
	if targets == nil {
		targets = []target{}
	}

	return targets, nil
}

// saveAceSpec は判定結果を保存する。条件を再確認して(取り下げ済み・既に埋まっているものは除く)
// 更新した件数を返す。
func saveAceSpec(db *gorm.DB, postId string, cardId string, cardName string, imageURL string, at time.Time) (int64, error) {
	tx := db.Exec(
		"UPDATE deck_code_posts SET ace_spec_card_id = ?, ace_spec_card_name = ?, ace_spec_image_url = ?, updated_at = ? "+
			"WHERE id = ? AND unpublished_at IS NULL AND ace_spec_card_id = ''",
		cardId, cardName, imageURL, at, postId,
	)

	return tx.RowsAffected, tx.Error
}

// refreshOgImage は個別ページを1回取得して、新しい内容の OGP 画像を作らせる。
// 画像はページの描画に付随して生成・アップロードされるため、本文は読み捨てる。
func refreshOgImage(ctx context.Context, client *http.Client, baseURL string, postId string) error {
	url := deckCodePostPageURL(baseURL, postId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("webapp responded with %d for %s", res.StatusCode, url)
	}

	return nil
}

// run は対象を1件ずつ判定し直して保存し、必要なら OGP 画像を作り直す。
func run(
	ctx context.Context,
	db *gorm.DB,
	deckCard repository.DeckCardInterface,
	ogpClient *http.Client,
	webappBaseURL string,
	opts options,
) (result, error) {
	var ret result

	targets, err := fetchTargets(db, opts)
	if err != nil {
		return ret, err
	}

	log.Printf("found %d posts without ace spec\n", len(targets))

	for i, t := range targets {
		if i > 0 {
			time.Sleep(requestInterval)
		}

		card, err := deckCard.FindAceSpec(ctx, t.Code)
		if err != nil {
			ret.failed++
			log.Printf("failed to look up ace spec: post=%s code=%s: %v\n", t.ID, t.Code, err)
			continue
		}
		if card == nil {
			ret.noAceSpec++
			log.Printf("no ace spec: post=%s deck=%s code=%s\n", t.ID, t.DeckName, t.Code)
			continue
		}

		if opts.dryRun {
			ret.updated++
			log.Printf("[dry-run] would fill: post=%s deck=%s code=%s ace_spec=%s(%s)\n",
				t.ID, t.DeckName, t.Code, card.CardName, card.CardId)
			continue
		}

		affected, err := saveAceSpec(db, t.ID, card.CardId, card.CardName, card.ImageURL, time.Now().Local())
		if err != nil {
			ret.failed++
			log.Printf("failed to save ace spec: post=%s: %v\n", t.ID, err)
			continue
		}
		if affected == 0 {
			// 実行中に取り下げられた・他の実行が先に埋めた
			log.Printf("skip (already filled or withdrawn): post=%s\n", t.ID)
			continue
		}

		ret.updated++
		log.Printf("filled: post=%s deck=%s code=%s ace_spec=%s(%s)\n",
			t.ID, t.DeckName, t.Code, card.CardName, card.CardId)

		if !opts.refreshOgp {
			continue
		}
		if err := refreshOgImage(ctx, ogpClient, webappBaseURL, t.ID); err != nil {
			ret.failed++
			log.Printf("failed to refresh ogp image: post=%s: %v\n", t.ID, err)
			continue
		}
		log.Printf("refreshed ogp image: post=%s\n", t.ID)
	}

	return ret, nil
}

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みも OGP 画像の生成も行わず対象の確認のみ行う")
	userId := flag.String("user-id", "", "対象を絞るユーザID")
	postId := flag.String("post-id", "", "対象を絞る投稿ID")
	limit := flag.Int("limit", 0, "1回の実行で扱う最大件数。0 なら制限しない")
	refreshOgp := flag.Bool("refresh-ogp", true, "true の場合、埋めた投稿の個別ページを取得して OGP 画像を作り直す")
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

	opts := options{
		dryRun:     *dryRun,
		userId:     *userId,
		postId:     *postId,
		limit:      *limit,
		refreshOgp: *refreshOgp,
	}

	webappBaseURL := webappBaseURLFromEnv()
	if *dryRun {
		log.Printf("[dry-run] checking deck code posts without ace spec. no changes will be made\n")
	} else {
		log.Printf("filling ace spec for deck code posts (refresh-ogp=%v, webapp=%s)\n", opts.refreshOgp, webappBaseURL)
	}

	ret, err := run(
		context.Background(),
		db,
		infrastructure.NewDeckCard(infrastructure.DeckCardAPIBaseURLFromEnv()),
		&http.Client{Timeout: ogpRefreshTimeout},
		webappBaseURL,
		opts,
	)
	if err != nil {
		log.Printf("failed to backfill ace spec: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	verb := "filled"
	if *dryRun {
		verb = "would be filled"
	}
	log.Printf("completed: %d posts %s, %d without ace spec, %d failed\n", ret.updated, verb, ret.noAceSpec, ret.failed)

	if ret.failed > 0 {
		os.Exit(ExitCodeNG)
	}

	os.Exit(ExitCodeOK)
}
