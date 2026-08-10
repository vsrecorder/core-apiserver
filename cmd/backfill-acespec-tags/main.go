// backfill-acespec-tags は、ACE SPECカードを全ユーザー共通の「プリセットタグ」として
// tags テーブルへ投入する初期化/更新バッチ。
//
// ACE SPECかどうかの判定情報源は cards テーブルのカード名で、ACE SPECカードには
// 必ず "(ACE SPEC)" の目印が付く(deckcard-api の app/core/constants.py と同じ規約)。
// ここからカード名を取り出し、目印を除いた名前でプリセットタグ(preset_flg=true,
// user_id='')を作る。プリセットは誰でも自分のデッキ/デッキコードに付与できるが、
// 編集・削除はできない。
//
// 対象は既定で現行スタンダードのレギュレーションマーク H のカードのみ(-regulation-mark)。
// 旧マークで刷られた同名の再録は除外し、現行の ACE SPEC だけをプリセットにする。
// レギュレーションが更新されたら -regulation-mark=I のように指定して再実行する。
//
// 冪等性: 既存のプリセットタグと名前で突き合わせ、未登録のものだけを追加する。
// 新しいACE SPECカードが cards に増えたら、このバッチを再実行すれば差分だけ投入される。
//
// 使い方:
//
//	# 投入対象を確認するだけ(デフォルト、書き込みなし。既定は regulation_mark=H)
//	go run ./cmd/backfill-acespec-tags
//
//	# 実際に tags へ投入する
//	go run ./cmd/backfill-acespec-tags -dry-run=false
//
//	# 対象のレギュレーションマークを変える
//	go run ./cmd/backfill-acespec-tags -regulation-mark=I -dry-run=false
package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
	ulid "github.com/oklog/ulid/v2"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// aceSpecSuffix は cards.card_name に付くACE SPECの目印。deckcard-api と揃える。
const aceSpecSuffix = "(ACE SPEC)"

// aceSpecTagColor はACE SPECプリセットタグの色。ACE SPECカードのマゼンタ調に合わせた色。
// タグは背景色＋白の太字で表示するため、白文字が読める濃さのマゼンタにしている。
const aceSpecTagColor = "#FF007F"

// maxTagNameLength は tags.name VARCHAR(32) に対応する上限。超える名前は投入しない。
const maxTagNameLength = 32

var entropy = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず投入対象の確認のみ行う")
	// 対象のレギュレーションマーク。既定は現行スタンダードの H のみ。
	// レギュレーションが更新されたら -regulation-mark=I のように指定する。
	regulationMark := flag.String("regulation-mark", "H", "対象とする cards.regulation_mark")
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

	// 1. cards から ACE SPEC カード名(目印付き)を取得する。
	// 対象は指定レギュレーションマーク(既定 H)のみ。旧マークで刷られた同名の再録は除外し、
	// 現行スタンダードの ACE SPEC だけをプリセットにする。
	var rawNames []string
	if tx := db.Raw(
		"SELECT DISTINCT card_name FROM cards WHERE card_name LIKE ? AND regulation_mark = ? ORDER BY card_name ASC",
		"%"+aceSpecSuffix+"%",
		*regulationMark,
	).Scan(&rawNames); tx.Error != nil {
		log.Printf("failed to query cards: %v\n", tx.Error)
		os.Exit(ExitCodeNG)
	}

	// 2. 目印を除いた一意のタグ名にする。
	seen := make(map[string]struct{}, len(rawNames))
	names := make([]string, 0, len(rawNames))
	for _, raw := range rawNames {
		name := strings.TrimSpace(strings.ReplaceAll(raw, aceSpecSuffix, ""))
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > maxTagNameLength {
			log.Printf("skip (name too long > %d chars): %s\n", maxTagNameLength, name)
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}

	// 3. 既存のプリセットタグと名前で突き合わせ、未登録は新規作成、
	//    既存でも色が変わっていれば色だけ更新する(色の定義を変えても再実行で反映される)。
	tagRepo := infrastructure.NewTag(db)
	existing, err := tagRepo.FindPresets(context.Background())
	if err != nil {
		log.Printf("failed to list preset tags: %v\n", err)
		os.Exit(ExitCodeNG)
	}
	existingByName := make(map[string]*entity.Tag, len(existing))
	for _, tag := range existing {
		existingByName[tag.Name] = tag
	}

	toCreate := make([]string, 0, len(names))
	toRecolor := make([]*entity.Tag, 0)
	for _, name := range names {
		if tag, ok := existingByName[name]; ok {
			if tag.Color != aceSpecTagColor {
				toRecolor = append(toRecolor, tag)
			}
			continue
		}
		toCreate = append(toCreate, name)
	}

	if *dryRun {
		log.Printf("[dry-run] regulation_mark=%s: ACE SPECカード名=%d件, 既存プリセット=%d件, 新規投入=%d件, 色更新=%d件 (書き込みは行いません)\n",
			*regulationMark, len(names), len(existing), len(toCreate), len(toRecolor))
		for _, name := range toCreate {
			log.Printf("[dry-run] would create preset tag: %s\n", name)
		}
		for _, tag := range toRecolor {
			log.Printf("[dry-run] would recolor preset tag: %s (%s -> %s)\n", tag.Name, tag.Color, aceSpecTagColor)
		}
		os.Exit(ExitCodeOK)
	}

	now := time.Now().Local()
	created := 0
	for _, name := range toCreate {
		id, err := generateId()
		if err != nil {
			log.Printf("failed to generate id for %s: %v\n", name, err)
			continue
		}

		// プリセットタグ: user_id='' / preset_flg=true。
		tag := entity.NewTag(id, now, now, "", name, aceSpecTagColor, true)
		if err := tagRepo.Save(context.Background(), tag); err != nil {
			log.Printf("failed to save preset tag %s: %v\n", name, err)
			continue
		}
		created++
	}

	recolored := 0
	for _, tag := range toRecolor {
		// 既存の id / created_at は保ち、色と updated_at だけ更新する。
		updated := entity.NewTag(tag.ID, tag.CreatedAt, now, "", tag.Name, aceSpecTagColor, true)
		if err := tagRepo.Save(context.Background(), updated); err != nil {
			log.Printf("failed to recolor preset tag %s: %v\n", tag.Name, err)
			continue
		}
		recolored++
	}

	log.Printf("completed: created %d, recolored %d preset tags (ACE SPEC, regulation_mark=%s, color=%s)\n",
		created, recolored, *regulationMark, aceSpecTagColor)
	os.Exit(ExitCodeOK)
}
