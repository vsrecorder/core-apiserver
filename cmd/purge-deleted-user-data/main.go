// purge-deleted-user-data は、退会したユーザ(users.deleted_at IS NOT NULL)に紐づくデータを
// DBから物理削除するためのツール。
//
// 退会処理(internal/usecase/user.go の User.Delete)は、論理削除(deleted_at)を持つテーブルは
// 論理削除、持たないテーブルは物理削除でユーザのデータを消す。そのため退会後も、記録・デッキ・
// 対戦などの実体は deleted_at が入った状態でDBに残っている。本ツールはその残っている行を
// 行ごと消し、DBから完全に取り除く。個人情報の削除要求への対応を想定している。
//
// 対象は -user-id で指定した1ユーザのみ。誤って有効なユーザのデータを消さないよう、
// 指定されたユーザが退会済みとして存在しない場合は何もせずに終了する。
//
// 削除する範囲:
//
//   - 「user_id を持つテーブル」と「それらへFKで繋がる中間テーブル」の全部。
//     論理削除済みかどうかは問わず、対象ユーザに紐づく行をすべて物理削除する。
//     対象テーブルの一覧は cmd/check-deleted-users-data の specs と1対1で対応する。
//
//   - users の行自体は削除しない。usecase.User.Create が IsWithdrawn で「退会済みのUIDでの
//     再登録」を拒否しており、行を消すとその防御が効かなくなるため。退会の記録としても残す。
//
//   - matches.opponents_user_id(他のユーザの対戦記録から対戦相手として参照されているもの)は
//     書き換えない。他人が作成したデータであり、退会処理も同じ扱いにしているため。
//
//   - 他のユーザの行であっても、対象ユーザのデッキ・記録・対戦を参照している行は
//     巻き込んで削除する(親を物理削除する以上、FK制約により先に消す必要があるため)。
//     巻き込む件数は実行前に警告として表示する。通常は0件になる。
//
// 冪等性: 2回目以降の実行は対象が0件になるだけで、結果は変わらない。
//
// 安全のための作り:
//
//   - 既定は -dry-run=true で、削除対象の件数を表示するだけでDBを変更しない。
//   - 実際に削除する場合も、実行前に対象の内訳を表示して yes の入力を求める(-yes で省略)。
//   - 削除は1つのトランザクションで行い、途中で失敗したらすべてロールバックする。
//   - 削除後に同じ条件で数え直し、1件でも残っていればロールバックして異常終了する。
//
// 使い方:
//
//	# 削除対象の件数を確認する(DBは変更しない)
//	go run ./cmd/purge-deleted-user-data -user-id <user_id>
//
//	# 実際に物理削除する(実行前に yes の入力を求める)
//	go run ./cmd/purge-deleted-user-data -user-id <user_id> -dry-run=false
//
//	# 確認を省略して削除する(バッチから実行する場合)
//	go run ./cmd/purge-deleted-user-data -user-id <user_id> -dry-run=false -yes
//
// 終了コード: 0 = 正常終了、1 = エラー(対象が退会済みでない・削除失敗など)
package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// 対象ユーザの親テーブルを表すサブクエリ。子テーブルには user_id を持たないものがあり、
// またFKの都合で親より先に消す必要があるため、「どの親を消すか」をここで1度だけ定義して
// 各テーブルの条件から参照する。@user_id には対象ユーザのIDが入る。
const (
	targetRecords = `SELECT id FROM records WHERE user_id = @user_id`
	targetDecks   = `SELECT id FROM decks WHERE user_id = @user_id`

	// deck_codes.deck_id は decks へのFK。他人が対象ユーザのデッキにデッキコードを作る導線は
	// 無いが、もし存在すると decks を物理削除できないため、デッキ経由のものも対象に含める。
	targetDeckCodes = `SELECT id FROM deck_codes WHERE user_id = @user_id OR deck_id IN (` + targetDecks + `)`

	// matches.record_id は records へのFK。user_id が食い違う不整合があっても records を
	// 消せるよう、記録経由のものも対象に含める。
	targetMatches = `SELECT id FROM matches WHERE user_id = @user_id OR record_id IN (` + targetRecords + `)`
)

// tableSpec は1テーブル分の物理削除の定義。
type tableSpec struct {
	// name は対象のテーブル名。
	name string

	// where は対象ユーザに紐づく行を選ぶ条件。件数の確認と削除の両方でこの条件を使うため、
	// 「数えた行」と「消す行」が食い違わない。@user_id に対象ユーザのIDが入る。
	where string

	// foreignWhere は where のうち、対象ユーザ以外のユーザの行を選ぶ条件。
	// 対象ユーザのデッキ・記録・対戦を参照しているために巻き込んで消えてしまう行を、
	// 実行前の警告として数えるために使う。巻き込みが起きえないテーブルは空文字。
	foreignWhere string

	// note は件数表示に添える補足(なぜこのテーブルが対象なのかが分かるようにするためのもの)。
	note string
}

// countQuery は対象ユーザに紐づく行数を数えるSQLを返す。
func (s *tableSpec) countQuery() string {
	return `SELECT COUNT(*) FROM ` + s.name + ` WHERE ` + s.where
}

// deleteQuery は対象ユーザに紐づく行を物理削除するSQLを返す。
func (s *tableSpec) deleteQuery() string {
	return `DELETE FROM ` + s.name + ` WHERE ` + s.where
}

// foreignCountQuery は巻き込んで削除される他ユーザの行数を数えるSQLを返す。
func (s *tableSpec) foreignCountQuery() string {
	return `SELECT COUNT(*) FROM ` + s.name + ` WHERE ` + s.foreignWhere
}

// specs は削除対象のテーブル定義。**この並び順のまま削除する**。
//
// 物理削除はFK制約に阻まれるため、子テーブル → 親テーブルの順に並べてある。
// 具体的には次の依存があり、入れ替えるとFK違反でトランザクションごと失敗する:
//
//	match_tags / match_pokemon_sprites / games → matches → records
//	deck_code_tags → deck_codes → decks
//	deck_tags / deck_pokemon_sprites / user_favorite_decks → decks
//	record_tags → records
//	deck_tags / deck_code_tags / record_tags / match_tags → tags
//
// 対象テーブルの一覧は cmd/check-deleted-users-data の specs と1対1で対応する
// (あちらは残存確認、こちらは物理削除)。ユーザに紐づくテーブルを追加したときは両方に足すこと。
var specs = []tableSpec{
	// --- 対戦(matches)の子。matches より先に消す ---
	{
		name:  "match_tags",
		where: `match_id IN (` + targetMatches + `)`,
		note:  "対戦に付けたタグ。user_id を持たないため対戦経由でたどる",
	},
	{
		name:  "match_pokemon_sprites",
		where: `match_id IN (` + targetMatches + `)`,
		note:  "対戦相手のデッキのスプライト。user_id を持たないため対戦経由でたどる",
	},
	{
		name:         "games",
		where:        `user_id = @user_id OR match_id IN (` + targetMatches + `)`,
		foreignWhere: `user_id <> @user_id AND match_id IN (` + targetMatches + `)`,
		note:         "対局。user_id 直と、対象の対戦にぶら下がるものの両方",
	},

	// --- 記録(records)の子。records より先に消す ---
	{
		name:         "matches",
		where:        `user_id = @user_id OR record_id IN (` + targetRecords + `)`,
		foreignWhere: `user_id <> @user_id AND record_id IN (` + targetRecords + `)`,
		note:         "対戦。user_id 直と、対象の記録にぶら下がるものの両方",
	},
	{
		name:  "record_tags",
		where: `record_id IN (` + targetRecords + `)`,
		note:  "記録に付けたタグ。user_id を持たないため記録経由でたどる",
	},
	{
		name:  "records",
		where: `user_id = @user_id`,
		note:  "記録",
	},

	// --- デッキ(decks)の子。decks より先に消す ---
	{
		name:  "deck_code_tags",
		where: `deck_code_id IN (` + targetDeckCodes + `)`,
		note:  "デッキコードに付けたタグ。user_id を持たないためデッキコード経由でたどる",
	},
	{
		name:         "deck_codes",
		where:        `user_id = @user_id OR deck_id IN (` + targetDecks + `)`,
		foreignWhere: `user_id <> @user_id AND deck_id IN (` + targetDecks + `)`,
		note:         "デッキコード。user_id 直と、対象のデッキにぶら下がるものの両方",
	},
	{
		name:  "deck_tags",
		where: `deck_id IN (` + targetDecks + `)`,
		note:  "デッキに付けたタグ。user_id を持たないためデッキ経由でたどる",
	},
	{
		name:  "deck_pokemon_sprites",
		where: `deck_id IN (` + targetDecks + `)`,
		note:  "デッキのスプライト。user_id を持たないためデッキ経由でたどる",
	},
	{
		name:         "user_favorite_decks",
		where:        `user_id = @user_id OR deck_id IN (` + targetDecks + `)`,
		foreignWhere: `user_id <> @user_id AND deck_id IN (` + targetDecks + `)`,
		note:         "お気に入り。対象ユーザが付けたものと、対象ユーザのデッキに他人が付けたものの両方",
	},
	{
		name:  "decks",
		where: `user_id = @user_id`,
		note:  "デッキ",
	},

	// --- タグ。各種 *_tags から参照されるため、それらより後に消す ---
	{
		name: "tags",
		// プリセットタグ(user_id = '')は全ユーザ共通で誰のものでもない。@user_id が空文字だと
		// それを消してしまうため、対象ユーザが退会済みユーザとして存在することを確認してから
		// 実行する(users に空文字IDの行は存在しないため、空文字では対象なしになる)。
		where: `user_id = @user_id`,
		note:  "自分で作ったタグ。プリセット(user_id が空)は対象外",
	},

	// --- user_id を直接持つ、他から参照されないテーブル ---
	{
		name:  "unofficial_events",
		where: `user_id = @user_id`,
		note:  "自由形式のイベント",
	},
	{
		name:  "users_players",
		where: `user_id = @user_id`,
		note:  "プレイヤーIDの紐付け",
	},
	{
		name:  "user_streaks",
		where: `user_id = @user_id`,
		note:  "週次ストリークの状態",
	},
	{
		name:  "user_daily_activities",
		where: `user_id = @user_id`,
		note:  "日別の活動記録",
	},
	{
		name:  "user_badges",
		where: `user_id = @user_id`,
		note:  "獲得済みのバッジ",
	},
	{
		name:  "user_environment_badges",
		where: `user_id = @user_id`,
		note:  "獲得済みの環境バッジ",
	},
	{
		name:  "notifications",
		where: `user_id = @user_id`,
		note:  "アプリ内通知",
	},
	{
		name:  "push_subscriptions",
		where: `user_id = @user_id`,
		note:  "Web Push の購読",
	},
	{
		name:  "push_deliveries",
		where: `user_id = @user_id`,
		note:  "Web Push の配信履歴",
	},
	{
		name:  "user_acquisitions",
		where: `user_id = @user_id`,
		note:  "流入経路",
	},
	{
		name:  "user_gyms",
		where: `user_id = @user_id`,
		note:  "Myジム",
	},
}

// tableCount は1テーブル分の件数。
type tableCount struct {
	table string
	count int64
	note  string
}

// targetUser は削除対象のユーザ。
type targetUser struct {
	ID        string
	Name      string
	CreatedAt string
	DeletedAt string
}

func main() {
	targetUserId := flag.String("user-id", "", "物理削除の対象にする退会済みユーザのID(必須)")
	dryRun := flag.Bool("dry-run", true, "true の場合、削除は行わず対象件数の確認のみ行う")
	yes := flag.Bool("yes", false, "削除前の確認を省略する")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	// 対象を1ユーザに限定する。未指定を「全ユーザ」と解釈すると、取り違えたときの被害が
	// 全退会ユーザに及ぶため、必ず明示させる。
	if *targetUserId == "" {
		log.Printf("-user-id は必須です(物理削除の対象にする退会済みユーザのIDを指定してください)\n")
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
		log.Printf("failed to connect database: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	// 退会済みであることを削除の前提条件にする。有効なユーザのデータを消してしまうと
	// 復旧できないため、ここで確実に弾く。
	user, err := findDeletedUser(db, *targetUserId)
	if err != nil {
		log.Printf("failed to fetch user: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	if user == nil {
		log.Printf("対象が見つかりません(user_id=%s は存在しないか、まだ退会していません)\n", *targetUserId)
		os.Exit(ExitCodeNG)
	}

	log.Printf("対象ユーザ: user_id=%s name=%s created_at=%s deleted_at=%s\n",
		user.ID, displayName(user.Name), user.CreatedAt, user.DeletedAt)

	counts, err := countAll(db, *targetUserId)
	if err != nil {
		log.Printf("failed to count target rows: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	foreignCounts, err := countForeignAll(db, *targetUserId)
	if err != nil {
		log.Printf("failed to count rows of other users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	reportCounts(counts, foreignCounts)

	if total(counts) == 0 {
		log.Printf("物理削除するデータはありません\n")
		os.Exit(ExitCodeOK)
	}

	if *dryRun {
		log.Printf("dry-run のため削除しません(実際に削除するには -dry-run=false を指定してください)\n")
		os.Exit(ExitCodeOK)
	}

	// users の行は残す。usecase.User.Create の IsWithdrawn による再登録拒否を効かせ続けるため。
	log.Printf("users の行(user_id=%s)は退会済みのまま残します\n", user.ID)

	if !*yes && !confirm(fmt.Sprintf("user_id=%s のデータ %d 件を物理削除します。元に戻せません。", user.ID, total(counts))) {
		log.Printf("削除を中止しました\n")
		os.Exit(ExitCodeOK)
	}

	results, err := purge(db, *targetUserId)
	if err != nil {
		log.Printf("failed to purge: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	for _, r := range results {
		log.Printf("  %-24s %6d 件 削除しました\n", r.table, r.count)
	}

	log.Printf("合計 %d 件を物理削除しました\n", total(results))

	os.Exit(ExitCodeOK)
}

// findDeletedUser は退会済みのユーザを1件取得する。存在しない、または退会していない場合は
// nil を返す。gorm.DeletedAt による論理削除は既定のクエリから除外されるため Unscoped で引く。
func findDeletedUser(db *gorm.DB, userId string) (*targetUser, error) {
	var rows []*model.User
	if tx := db.Unscoped().Model(&model.User{}).
		Where("id = ? AND deleted_at IS NOT NULL", userId).
		Find(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	if len(rows) == 0 {
		return nil, nil
	}

	row := rows[0]

	return &targetUser{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Format(timeLayout),
		DeletedAt: row.DeletedAt.Time.Format(timeLayout),
	}, nil
}

// timeLayout は表示に使う日時の書式。
const timeLayout = "2006-01-02T15:04:05Z07:00"

// displayName は表示用のユーザ名。users.name は任意項目で、未設定(NULL)だと空文字になるため、
// 空欄で表示されないようプレースホルダに置き換える。
func displayName(name string) string {
	if name == "" {
		return "名前未設定"
	}

	return name
}

// countAll は全テーブル分の削除対象件数を、テーブル定義(= 削除)の順に返す。
// 0件のテーブルは表示しても意味がないため含めない。
func countAll(db *gorm.DB, userId string) ([]tableCount, error) {
	var counts []tableCount

	for _, spec := range specs {
		count, err := countRows(db, spec.countQuery(), userId)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.name, err)
		}

		if count > 0 {
			counts = append(counts, tableCount{table: spec.name, count: count, note: spec.note})
		}
	}

	return counts, nil
}

// countForeignAll は、対象ユーザのデッキ・記録・対戦を参照しているために巻き込んで削除される
// 他ユーザの行数を返す。通常は0件で、0件でないものだけを返す。
func countForeignAll(db *gorm.DB, userId string) ([]tableCount, error) {
	var counts []tableCount

	for _, spec := range specs {
		if spec.foreignWhere == "" {
			continue
		}

		count, err := countRows(db, spec.foreignCountQuery(), userId)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.name, err)
		}

		if count > 0 {
			counts = append(counts, tableCount{table: spec.name, count: count, note: spec.note})
		}
	}

	return counts, nil
}

// countRows は件数を数えるSQLを実行する。
func countRows(db *gorm.DB, query string, userId string) (int64, error) {
	var count int64
	if tx := db.Raw(query, namedUserId(userId)).Scan(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return count, nil
}

// namedUserId は各SQLの @user_id へ対象ユーザのIDを渡すための名前付き引数。
// 1つの条件のなかに @user_id が何度も現れる(親のサブクエリを埋め込むため)ので、
// 位置プレースホルダでは出現回数ぶんの引数を数えて渡すことになり、取り違えやすい。
func namedUserId(userId string) sql.NamedArg {
	return sql.Named("user_id", userId)
}

// total は件数の合計。
func total(counts []tableCount) int64 {
	var sum int64
	for _, c := range counts {
		sum += c.count
	}

	return sum
}

// reportCounts は削除対象の内訳と、巻き込んで削除される他ユーザの行の警告を表示する。
func reportCounts(counts []tableCount, foreignCounts []tableCount) {
	if len(counts) == 0 {
		return
	}

	log.Printf("物理削除の対象:\n")
	for _, c := range counts {
		log.Printf("  %-24s %6d 件  %s\n", c.table, c.count, c.note)
	}
	log.Printf("  合計 %d 件\n", total(counts))

	for _, c := range foreignCounts {
		log.Printf("警告: %s の他ユーザの行 %d 件を巻き込んで削除します(対象ユーザのデッキ・記録・対戦を参照しているため)\n",
			c.table, c.count)
	}
}

// purge は対象ユーザに紐づく行を物理削除し、テーブルごとの削除件数を返す。
//
// 削除はすべて1つのトランザクションで行う。テーブルをまたいで中途半端に消えると、
// どこまで消えたのかが分からないまま残骸だけが残るため。
func purge(db *gorm.DB, userId string) ([]tableCount, error) {
	var results []tableCount

	err := db.Transaction(func(tx *gorm.DB) error {
		// ロールバックした場合に途中までの結果が残らないよう、実行のたびに作り直す
		results = nil

		ret, err := purgeInTx(tx, userId)
		if err != nil {
			return err
		}

		results = ret

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// purgeInTx は削除本体。呼び出し元のトランザクションの中で実行する。
// 削除後に同じ条件で数え直し、1件でも残っていればエラーを返してロールバックさせる。
func purgeInTx(tx *gorm.DB, userId string) ([]tableCount, error) {
	var results []tableCount

	// specs の並び順は子テーブル → 親テーブルのFK依存を表している。順に実行すること。
	for _, spec := range specs {
		ret := tx.Exec(spec.deleteQuery(), namedUserId(userId))
		if ret.Error != nil {
			return nil, fmt.Errorf("%s: %w", spec.name, ret.Error)
		}

		if ret.RowsAffected > 0 {
			results = append(results, tableCount{table: spec.name, count: ret.RowsAffected, note: spec.note})
		}
	}

	// 検算。削除の条件と件数の条件は同じ where を使っているため、ここで残っているとしたら
	// 削除が実際には効いていない(トリガや権限など想定外の要因)ことになる。
	// 中途半端な状態でコミットしないよう、エラーにしてロールバックさせる。
	for _, spec := range specs {
		count, err := countRows(tx, spec.countQuery(), userId)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.name, err)
		}

		if count > 0 {
			return nil, fmt.Errorf("%s: 削除したはずのデータが %d 件残っています", spec.name, count)
		}
	}

	return results, nil
}

// confirm は標準入力から yes の入力を求める。yes 以外の入力・入力の失敗はすべて中止として扱う。
func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s 続ける場合は yes と入力してください: ", prompt)

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}

	return strings.TrimSpace(line) == "yes"
}
