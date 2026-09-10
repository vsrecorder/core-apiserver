// backfill-acespec-tags は、ACE SPECカードを全ユーザー共通の「プリセットタグ」として
// tags テーブルへ投入する初期化/更新バッチ。
//
// ACE SPECかどうかの判定情報源は cards テーブルのカード名で、ACE SPECカードには
// 必ず "(ACE SPEC)" の目印が付く(deckcard-api の app/core/constants.py と同じ規約)。
// ここからカード名を取り出し、目印を除いた名前でプリセットタグ(preset_flg=true,
// user_id=”)を作る。プリセットは誰でも自分のデッキ/デッキコードに付与できるが、
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
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
	ulid "github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

const appName = "backfill-acespec-tags"

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

// ULID を単調増加(モノトニック)で発番する。生成順(=card id 昇順)が id の昇順に一致するため、
// FindPresets を id 昇順で引けばプリセットが card id 昇順(≒収録順)で並ぶ。
var entropy = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}

// fetchAceSpecCardNames は cards から、指定レギュレーションマークの ACE SPEC カード名(目印付き)を
// card id が小さい順で取得する。同名の ACE SPEC が複数カード(再録含む)に跨ることがあるため、
// 名前ごとの最小 id で並び順を決める。
func fetchAceSpecCardNames(db *gorm.DB, regulationMark string) ([]string, error) {
	var rawNames []string
	if tx := db.Raw(
		"SELECT card_name FROM cards WHERE card_name LIKE ? AND regulation_mark = ? GROUP BY card_name ORDER BY MIN(id) ASC",
		"%"+aceSpecSuffix+"%",
		regulationMark,
	).Scan(&rawNames); tx.Error != nil {
		return nil, tx.Error
	}
	return rawNames, nil
}

// cleanAceSpecNames は目印(aceSpecSuffix)を除き、空・長すぎる名前を除外し、重複を取り除く。
// 入力の並び順(=card id 昇順)は保持する。
func cleanAceSpecNames(rawNames []string) []string {
	seen := make(map[string]struct{}, len(rawNames))
	names := make([]string, 0, len(rawNames))
	for _, raw := range rawNames {
		name := strings.TrimSpace(strings.ReplaceAll(raw, aceSpecSuffix, ""))
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > maxTagNameLength {
			slog.Warn("skipping card name: longer than the tag name limit",
				slog.Int("max_chars", maxTagNameLength), slog.String("card_name", name))
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず投入対象の確認のみ行う")
	// 対象のレギュレーションマーク。既定は現行スタンダードの H のみ。
	// レギュレーションが更新されたら -regulation-mark=I のように指定する。
	regulationMark := flag.String("regulation-mark", "H", "対象とする cards.regulation_mark")
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

	// 1. cards から ACE SPEC カード名(目印付き)を card id 昇順で取得する。
	rawNames, err := fetchAceSpecCardNames(db, *regulationMark)
	if err != nil {
		slog.Error("failed to query cards", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	// 2. 目印を除いた一意のタグ名にする(取得順=card id 昇順を保つ)。
	names := cleanAceSpecNames(rawNames)

	// 3. 既存のプリセットタグと名前で突き合わせ、未登録は新規作成、
	//    既存でも色が変わっていれば色だけ更新する(色の定義を変えても再実行で反映される)。
	//    突き合わせ対象は ACE SPEC の群だけ(大会順位など他の群とは名前空間が別)。
	tagRepo := infrastructure.NewTag(db)
	existing, err := tagRepo.FindPresets(context.Background(), entity.TagPresetCategoryAceSpec)
	if err != nil {
		slog.Error("failed to list preset tags", logging.Err(err))
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
		slog.Info("planned preset tag changes",
			slog.String("regulation_mark", *regulationMark),
			slog.Int("ace_spec_card_names", len(names)),
			slog.Int("existing_presets", len(existing)),
			slog.Int("to_create", len(toCreate)),
			slog.Int("to_recolor", len(toRecolor)),
			slog.Bool("dry_run", true),
		)
		for _, name := range toCreate {
			slog.Info("preset tag to create", slog.String("name", name), slog.Bool("dry_run", true))
		}
		for _, tag := range toRecolor {
			slog.Info("preset tag to recolor",
				slog.String("name", tag.Name),
				slog.String("from_color", tag.Color), slog.String("to_color", aceSpecTagColor),
				slog.Bool("dry_run", true),
			)
		}
		os.Exit(ExitCodeOK)
	}

	now := time.Now().Local()
	created := 0
	for _, name := range toCreate {
		id, err := generateId()
		if err != nil {
			slog.Error("failed to generate id", slog.String("name", name), logging.Err(err))
			continue
		}

		// プリセットタグ: user_id='' / preset_flg=true / 群は ACE SPEC。
		tag := entity.NewTag(id, now, now, "", name, aceSpecTagColor, true, entity.TagPresetCategoryAceSpec, "")
		if err := tagRepo.Save(context.Background(), tag); err != nil {
			slog.Error("failed to save preset tag", slog.String("name", name), logging.Err(err))
			continue
		}
		created++
	}

	recolored := 0
	for _, tag := range toRecolor {
		// 既存の id / created_at / 文字色は保ち、背景色と updated_at だけ更新する。
		// Save は全カラムを書くため、ここで持ち回らない値は空で上書きされてしまう。
		updated := entity.NewTag(tag.ID, tag.CreatedAt, now, "", tag.Name, aceSpecTagColor, true, entity.TagPresetCategoryAceSpec, tag.TextColor)
		if err := tagRepo.Save(context.Background(), updated); err != nil {
			slog.Error("failed to recolor preset tag", slog.String("name", tag.Name), logging.Err(err))
			continue
		}
		recolored++
	}

	slog.Info("completed",
		slog.Int("created", created),
		slog.Int("recolored", recolored),
		slog.String("regulation_mark", *regulationMark),
		slog.String("color", aceSpecTagColor),
	)
	os.Exit(ExitCodeOK)
}
