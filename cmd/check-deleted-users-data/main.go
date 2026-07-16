// check-deleted-users-data は、退会したユーザ(users.deleted_at IS NOT NULL)が作成した
// データが残っていないかを確認し、必要であれば削除するためのツール。
//
// 退会処理(internal/usecase/user.go の User.Delete)は、ユーザ本体を論理削除する前に
// 対戦記録・デッキ・デッキコード・プレイヤーID紐付けを連鎖削除している。本ツールは
// その結果を DB 側から検算し、消えているはずのデータが残っていないかを検出する。
//
// 検出結果は、原因と取るべき対処が異なるため次の3つに分類して表示する:
//
//   - 削除漏れ(NG)    … 退会処理が削除対象としているのに残っているデータ。退会処理が途中で
//     失敗した、もしくは退会処理の実装後に作られた経路(バッチ等)で
//     データが作られた可能性がある。調査と削除が必要。
//
//   - 未対応(WARN)    … そもそも退会処理が削除対象にしていないテーブルに残っているデータ。
//     バグではなく仕様上の未対応であり、根本的に対処するなら退会処理の
//     実装を変更する必要がある。実行するたびに必ず検出される点に注意。
//
//   - 参照(INFO)      … 他のユーザが作成したデータから、退会したユーザが参照されているもの。
//     他人のデータなので消してはいけない。異常ではない。
//
// 「残っている」の判定は、論理削除(gorm.DeletedAt)を持つテーブルは deleted_at IS NULL の
// 行、持たないテーブルは行の存在そのもの、としている。
//
// 既定では確認のみで、DB に一切書き込みを行わない。-delete を指定した場合に限り削除する。
// 削除は退会処理の実装(User.Delete)と揃え、論理削除を持つテーブルは deleted_at を入れ、
// 持たないテーブルは行を物理削除する。参照(INFO)は他人のデータのため -delete でも削除しない。
//
// 使い方:
//
//	# 退会ユーザ全体のサマリを表示する(確認のみ。DBは変更しない)
//	go run ./cmd/check-deleted-users-data
//
//	# どのユーザにどれだけ残っているかの内訳も表示する
//	go run ./cmd/check-deleted-users-data -verbose
//
//	# 特定の退会ユーザだけを確認する
//	go run ./cmd/check-deleted-users-data -user <user_id>
//
//	# 削除漏れがあった場合に終了コード 1 を返す(CI・定期実行で検知したい場合)
//	go run ./cmd/check-deleted-users-data -exit-code
//
//	# 削除漏れ(NG)を削除する(実行前に対象を表示して確認を求める)
//	go run ./cmd/check-deleted-users-data -delete
//
//	# 未対応(WARN)のデータもあわせて削除する
//	go run ./cmd/check-deleted-users-data -delete -include-unhandled
//
//	# 確認を省略して削除する(バッチから実行する場合)
//	go run ./cmd/check-deleted-users-data -delete -yes
//
// -include-unhandled は、退会処理が削除対象にしていないデータ(バッジ・通知・ストリーク等)まで
// 消すため、退会後に残す仕様だったものも消える点に注意する。まず -delete のみで実行し、
// 意図して消したい場合だけ指定すること。
//
// -exit-code が終了コード1にするのは「削除漏れ(NG)」があった場合のみで、「未対応(WARN)」は
// 対象にしない。未対応は退会処理の仕様上つねに残るため、これを異常扱いにすると定期実行が
// 鳴りっぱなしになるため。
//
// 終了コード: 0 = 正常終了(-exit-code 指定時は削除漏れなし)、1 = エラー(または削除漏れあり)
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
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

// category は検出したデータの分類。原因と取るべき対処が異なるため区別する。
type category int

const (
	// categoryLeak は退会処理が削除対象としているのに残っているデータ(削除漏れ)。
	categoryLeak category = iota
	// categoryUnhandled は退会処理が削除対象にしていないテーブルに残っているデータ(仕様上の未対応)。
	categoryUnhandled
	// categoryReference は他のユーザのデータから退会したユーザが参照されているもの(異常ではない)。
	categoryReference
)

// tableSpec は「退会したユーザのデータが残っていないか」を1テーブル分確認・削除するための定義。
type tableSpec struct {
	// name は表示に使う対象の名前。参照する列がテーブル内で一意に決まらない場合は
	// "matches.opponents_user_id" のように列名まで含める。
	name string

	// category は残っていた場合の分類。
	category category

	// query は退会ユーザごとの残存件数を返すSQL。(user_id, count) の2列を返すこと。
	// テーブルごとに論理削除の有無や、user_id へのたどり方(直接持つ / matches 経由 等)が
	// 違うため、共通化せずテーブルごとに書き下している。
	query string

	// deleteQuery は残っているデータを削除するSQL。query と同じ行が対象になるよう、
	// 絞り込み条件を query と一致させること。論理削除を持つテーブルは deleted_at を入れ、
	// 持たないテーブルは行を物理削除する(退会処理の実装に合わせている)。
	//
	// 空文字の場合、そのテーブルは -delete を指定しても削除しない。他のユーザが作成した
	// データからの参照(categoryReference)は消してはいけないため、削除するSQLを持たせない。
	deleteQuery string

	// ownerColumn は deleteQuery 内で退会ユーザのIDを指す列(例: "t.user_id")。
	// -user でユーザを絞り込む際に deleteQuery へ条件を足すために使う。
	ownerColumn string

	// note は表示に添える補足(検出された場合に原因の見当がつくようにするためのもの)。
	note string
}

// specs は確認対象のテーブル定義。退会処理(internal/usecase/user.go の User.Delete)が
// 何を削除しているかと1対1で対応させている。退会処理の実装や、user_id を持つテーブルを
// 追加・変更した場合は、ここも合わせて更新すること。
var specs = []tableSpec{
	// --- 退会処理が削除対象としているもの(残っていれば削除漏れ) ---
	{
		name:     "records",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM records t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		deleteQuery: `UPDATE records t SET deleted_at = now() FROM users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL AND t.deleted_at IS NULL`,
		ownerColumn: "t.user_id",
		note:        "User.Delete が user_id で洗い出して削除する",
	},
	{
		name:     "matches",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM matches t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		deleteQuery: `UPDATE matches t SET deleted_at = now() FROM users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL AND t.deleted_at IS NULL`,
		ownerColumn: "t.user_id",
		note:        "records の削除に連鎖して削除される。record を経由しない孤立行があると残る",
	},
	{
		name:     "games",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM games t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		deleteQuery: `UPDATE games t SET deleted_at = now() FROM users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL AND t.deleted_at IS NULL`,
		ownerColumn: "t.user_id",
		note:        "matches の削除に連鎖して削除される。match を経由しない孤立行があると残る",
	},
	{
		name:     "decks",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM decks t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		deleteQuery: `UPDATE decks t SET deleted_at = now() FROM users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL AND t.deleted_at IS NULL`,
		ownerColumn: "t.user_id",
		note:        "User.Delete が user_id で洗い出して削除する",
	},
	{
		name:     "deck_codes",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM deck_codes t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		deleteQuery: `UPDATE deck_codes t SET deleted_at = now() FROM users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL AND t.deleted_at IS NULL`,
		ownerColumn: "t.user_id",
		note:        "User.Delete が user_id で洗い出して削除する",
	},
	{
		name:     "users_players",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM users_players t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		deleteQuery: `UPDATE users_players t SET deleted_at = now() FROM users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL AND t.deleted_at IS NULL`,
		ownerColumn: "t.user_id",
		note:        "User.Delete が FindByUserId で引いた1件を削除する。有効な行が2件以上あると残る",
	},
	{
		name:     "unofficial_events",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM unofficial_events t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		// Record.Delete は records.unofficial_event_id をたどって削除するため、どの record からも
		// 参照されていない自由形式イベントは退会処理では消えない。ここでは user_id から直接消す。
		deleteQuery: `UPDATE unofficial_events t SET deleted_at = now() FROM users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL AND t.deleted_at IS NULL`,
		ownerColumn: "t.user_id",
		note:        "record 経由でのみ削除される。どの record からも参照されていないものは残る",
	},

	// --- 退会処理が削除対象にしていないもの(残っていても実装どおり) ---
	// これらは論理削除を持たないため、削除する場合は行ごと物理削除するしかない。
	{
		name:     "user_streaks",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM user_streaks t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		deleteQuery: `DELETE FROM user_streaks t USING users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL`,
		ownerColumn: "t.user_id",
		note:        "退会処理の対象外。論理削除を持たないため行ごと残る",
	},
	{
		name:     "user_badges",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM user_badges t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		deleteQuery: `DELETE FROM user_badges t USING users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL`,
		ownerColumn: "t.user_id",
		note:        "退会処理の対象外。論理削除を持たないため行ごと残る",
	},
	{
		name:     "user_environment_badges",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM user_environment_badges t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		deleteQuery: `DELETE FROM user_environment_badges t USING users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL`,
		ownerColumn: "t.user_id",
		note:        "退会処理の対象外。論理削除を持たないため行ごと残る",
	},
	{
		name:     "notifications",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM notifications t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		deleteQuery: `DELETE FROM notifications t USING users u
		              WHERE u.id = t.user_id AND u.deleted_at IS NOT NULL`,
		ownerColumn: "t.user_id",
		note:        "退会処理の対象外。論理削除を持たないため行ごと残る",
	},
	{
		name:     "match_pokemon_sprites",
		category: categoryUnhandled,
		// user_id を持たないため matches を経由してたどる。matches 自体が論理削除済みでも
		// スプライトの行は残るため、matches.deleted_at では絞り込まない。
		query: `SELECT m.user_id, COUNT(*) FROM match_pokemon_sprites t
		        JOIN matches m ON m.id = t.match_id
		        JOIN users u ON u.id = m.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY m.user_id`,
		deleteQuery: `DELETE FROM match_pokemon_sprites t USING matches m, users u
		              WHERE m.id = t.match_id AND u.id = m.user_id AND u.deleted_at IS NOT NULL`,
		ownerColumn: "m.user_id",
		note:        "退会処理の対象外。match が論理削除されてもスプライトの行は残る",
	},
	{
		name:     "deck_pokemon_sprites",
		category: categoryUnhandled,
		query: `SELECT d.user_id, COUNT(*) FROM deck_pokemon_sprites t
		        JOIN decks d ON d.id = t.deck_id
		        JOIN users u ON u.id = d.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY d.user_id`,
		deleteQuery: `DELETE FROM deck_pokemon_sprites t USING decks d, users u
		              WHERE d.id = t.deck_id AND u.id = d.user_id AND u.deleted_at IS NOT NULL`,
		ownerColumn: "d.user_id",
		note:        "退会処理の対象外。deck が論理削除されてもスプライトの行は残る",
	},

	// --- 他のユーザのデータからの参照(異常ではない) ---
	{
		name:     "matches.opponents_user_id",
		category: categoryReference,
		// 退会したユーザ「が」作成したデータではなく、他のユーザが作成した対戦記録の中で
		// 対戦相手として参照されているもの。他人のデータなので削除するSQLを持たせない。
		query: `SELECT t.opponents_user_id, COUNT(*) FROM matches t
		        JOIN users u ON u.id = t.opponents_user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.opponents_user_id`,
		note: "他のユーザの対戦記録から対戦相手として参照されているもの。消してはいけない",
	},
}

// finding は「退会したユーザのデータが1件以上残っている」ことを表す1件の検出結果。
type finding struct {
	table    string
	category category
	userId   string
	count    int64
	note     string
}

// deleteResult は1テーブル分の削除結果。
type deleteResult struct {
	table string
	count int64
}

func main() {
	verbose := flag.Bool("verbose", false, "どの退会ユーザにどれだけ残っているかの内訳を表示する")
	userId := flag.String("user", "", "確認・削除の対象を特定の退会ユーザのIDに絞る(未指定なら退会ユーザ全件)")
	exitCode := flag.Bool("exit-code", false, "true の場合、削除漏れが見つかったら終了コード1で終了する")
	deleteFlag := flag.Bool("delete", false, "検出した削除漏れ(NG)を実際に削除する(未指定なら確認のみでDBは変更しない)")
	includeUnhandled := flag.Bool("include-unhandled", false, "-delete と併用し、未対応(WARN)のデータもあわせて削除する")
	yes := flag.Bool("yes", false, "-delete の実行前の確認を省略する")
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

	names, err := fetchDeletedUserNames(db, *userId)
	if err != nil {
		log.Printf("failed to fetch deleted users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	if len(names) == 0 {
		if *userId != "" {
			log.Printf("確認対象の退会ユーザが見つかりません(user_id=%s は存在しないか、まだ退会していません)\n", *userId)
			os.Exit(ExitCodeNG)
		}

		log.Printf("退会したユーザ(deleted_at IS NOT NULL)が1人もいないため、確認対象がありません\n")
		os.Exit(ExitCodeOK)
	}

	if *userId != "" {
		log.Printf("退会ユーザ user_id=%s (%s) のデータを確認します\n", *userId, displayName(names, *userId))
	} else {
		log.Printf("退会ユーザ %d 人のデータを確認します\n", len(names))
	}

	findings, err := collect(db, specs, *userId)
	if err != nil {
		log.Printf("failed to collect findings: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	report(findings, names, *verbose)

	if *deleteFlag {
		remaining, err := runDelete(db, findings, names, *userId, *includeUnhandled, *yes, *verbose)
		if err != nil {
			log.Printf("failed to delete: %v\n", err)
			os.Exit(ExitCodeNG)
		}

		findings = remaining
	}

	if *exitCode && len(filterByCategory(findings, categoryLeak)) > 0 {
		os.Exit(ExitCodeNG)
	}

	os.Exit(ExitCodeOK)
}

// runDelete は検出結果のうち削除対象のものを削除し、削除後にもう一度確認し直した検出結果を返す。
// 確認を求めて中止された場合は、削除を行わずに元の検出結果をそのまま返す。
func runDelete(
	db *gorm.DB,
	findings []finding,
	names map[string]string,
	userId string,
	includeUnhandled bool,
	yes bool,
	verbose bool,
) ([]finding, error) {
	targets := deleteTargets(specs, includeUnhandled)
	targetFindings := filterByTables(findings, targets)

	if len(targetFindings) == 0 {
		log.Printf("削除対象のデータはありません\n")
		return findings, nil
	}

	log.Printf("以下のデータを削除します\n")
	reportSection(targetFindings, names, verbose)

	if !includeUnhandled && len(filterByCategory(findings, categoryUnhandled)) > 0 {
		log.Printf("未対応(WARN)のデータは削除しません。あわせて削除する場合は -include-unhandled を指定してください\n")
	}

	if !yes && !confirm("本当に削除しますか?") {
		log.Printf("削除を中止しました\n")
		return findings, nil
	}

	results, err := deleteFindings(db, targets, userId)
	if err != nil {
		return nil, err
	}

	var total int64
	for _, r := range results {
		log.Printf("  %-26s %6d 件 削除しました\n", r.table, r.count)
		total += r.count
	}
	log.Printf("合計 %d 件を削除しました\n", total)

	// 削除後に消えているかを確認し直す。ここで削除漏れが残る場合、本ツールの削除条件が
	// 検出条件と食い違っているため、そのまま報告する。
	log.Printf("削除後の状態を確認します\n")

	remaining, err := collect(db, specs, userId)
	if err != nil {
		return nil, err
	}

	report(remaining, names, verbose)

	return remaining, nil
}

// deleteTargets は -delete で削除する対象のテーブル定義を返す。
// 削除するSQLを持たないもの(= 他のユーザのデータからの参照)は必ず対象外にする。
func deleteTargets(specs []tableSpec, includeUnhandled bool) []tableSpec {
	var targets []tableSpec

	for _, spec := range specs {
		if spec.deleteQuery == "" {
			continue
		}
		if spec.category == categoryUnhandled && !includeUnhandled {
			continue
		}

		targets = append(targets, spec)
	}

	return targets
}

// deleteFindings は対象テーブルのデータを削除し、テーブルごとの削除件数を返す。
// 途中で失敗した場合、ここまでの削除もすべてロールバックされる(退会処理と同じ扱いにするため)。
func deleteFindings(db *gorm.DB, targets []tableSpec, userId string) ([]deleteResult, error) {
	var results []deleteResult

	err := db.Transaction(func(tx *gorm.DB) error {
		// ロールバック時に途中までの結果が残らないよう、トランザクションのたびに作り直す
		results = nil

		for _, spec := range targets {
			query := spec.deleteQuery
			var args []any

			// deleteQuery は必ず WHERE を持つため、条件はそのまま AND で足せる
			if userId != "" {
				query += " AND " + spec.ownerColumn + " = ?"
				args = append(args, userId)
			}

			ret := tx.Exec(query, args...)
			if ret.Error != nil {
				return fmt.Errorf("%s: %w", spec.name, ret.Error)
			}

			if ret.RowsAffected > 0 {
				results = append(results, deleteResult{table: spec.name, count: ret.RowsAffected})
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
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

// fetchDeletedUserNames は確認対象の退会ユーザについて、user_id をキーにした名前を返す。
// userId を指定した場合は、そのユーザが退会済みとして存在すれば1件、そうでなければ0件を返す。
// 確認対象のユーザ数としても使うため、件数だけが欲しい場合も len() で足りる。
//
// users.name は退会(論理削除)時に消していないため、退会後も表示に使える。
// gorm.DeletedAt を持つモデルは論理削除された行がデフォルトで除外されてしまうため、
// Unscoped で退会済みを取得する。
func fetchDeletedUserNames(db *gorm.DB, userId string) (map[string]string, error) {
	query := db.Unscoped().Model(&model.User{}).Where("deleted_at IS NOT NULL")
	if userId != "" {
		query = query.Where("id = ?", userId)
	}

	var rows []*model.User
	if tx := query.Find(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Name
	}

	return names, nil
}

// displayName は表示用のユーザ名を返す。users.name は任意項目で未設定(NULL)のことがあり、
// その場合は空文字になるため、空欄で表示されないようプレースホルダに置き換える。
func displayName(names map[string]string, userId string) string {
	if name := names[userId]; name != "" {
		return name
	}

	return "名前未設定"
}

// collect は全テーブル分の確認を実行し、残っていたデータを検出結果として返す。
// 結果はテーブル定義の順、同一テーブル内では残存件数の多い順に並べる。
func collect(db *gorm.DB, specs []tableSpec, userId string) ([]finding, error) {
	var findings []finding

	for _, spec := range specs {
		rows, err := queryCounts(db, spec, userId)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.name, err)
		}

		findings = append(findings, rows...)
	}

	return findings, nil
}

// queryCounts は1テーブル分のSQLを実行し、退会ユーザごとの残存件数を検出結果に変換する。
func queryCounts(db *gorm.DB, spec tableSpec, userId string) ([]finding, error) {
	// 退会ユーザ全件が対象のときは spec.query をそのまま使い、-user 指定時は
	// その結果をサブクエリとして user_id で絞り込む。spec.query 側の列名や
	// 別名(t / m / d)の違いに影響されずに絞り込めるようにするため。
	query := spec.query
	var args []any
	if userId != "" {
		query = "SELECT * FROM (" + spec.query + ") AS counts(user_id, count) WHERE user_id = ?"
		args = append(args, userId)
	}

	rows, err := db.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []finding
	for rows.Next() {
		var id string
		var count int64
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}

		findings = append(findings, finding{
			table:    spec.name,
			category: spec.category,
			userId:   id,
			count:    count,
			note:     spec.note,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 残存件数の多いユーザほど調査の優先度が高いため件数の降順、同数ならIDの昇順で安定させる
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].count != findings[j].count {
			return findings[i].count > findings[j].count
		}
		return findings[i].userId < findings[j].userId
	})

	return findings, nil
}

// filterByCategory は指定した分類の検出結果だけを返す。
func filterByCategory(findings []finding, category category) []finding {
	var ret []finding
	for _, f := range findings {
		if f.category == category {
			ret = append(ret, f)
		}
	}

	return ret
}

// filterByTables は指定したテーブル定義に対応する検出結果だけを返す。
func filterByTables(findings []finding, specs []tableSpec) []finding {
	tables := make(map[string]bool, len(specs))
	for _, spec := range specs {
		tables[spec.name] = true
	}

	var ret []finding
	for _, f := range findings {
		if tables[f.table] {
			ret = append(ret, f)
		}
	}

	return ret
}

// summarizeByTable は検出結果をテーブル単位に集約し、テーブル名・残存件数の合計・
// 該当した退会ユーザ数・補足を、検出結果に現れた順(= テーブル定義の順)で返す。
func summarizeByTable(findings []finding) []tableSummary {
	var summaries []tableSummary
	indexes := make(map[string]int)

	for _, f := range findings {
		i, ok := indexes[f.table]
		if !ok {
			indexes[f.table] = len(summaries)
			summaries = append(summaries, tableSummary{table: f.table, note: f.note})
			i = len(summaries) - 1
		}

		summaries[i].count += f.count
		summaries[i].userCount++
	}

	return summaries
}

// tableSummary はテーブル単位に集約した検出結果。
type tableSummary struct {
	table     string
	count     int64
	userCount int
	note      string
}

// report は確認結果を標準出力へ出力する。names は user_id をキーにした退会ユーザの名前。
func report(findings []finding, names map[string]string, verbose bool) {
	leaks := filterByCategory(findings, categoryLeak)
	unhandled := filterByCategory(findings, categoryUnhandled)
	references := filterByCategory(findings, categoryReference)

	if len(leaks) == 0 {
		log.Printf("OK: 退会処理が削除対象としているデータは、すべて削除されています\n")
	} else {
		log.Printf("NG: 削除漏れ(退会処理が削除するはずのデータが残っています)\n")
		reportSection(leaks, names, verbose)
	}

	if len(unhandled) > 0 {
		log.Printf("WARN: 未対応(退会処理が削除対象にしていないデータが残っています)\n")
		log.Printf("      バグではなく仕様上の未対応です。消すなら退会処理(internal/usecase/user.go)の実装変更が必要です\n")
		reportSection(unhandled, names, verbose)
	}

	if len(references) > 0 {
		log.Printf("INFO: 参照(他のユーザが作成したデータから、退会したユーザが参照されています)\n")
		log.Printf("      他人のデータのため削除しません。異常ではありません\n")
		reportSection(references, names, verbose)
	}

	if !verbose && len(findings) > 0 {
		log.Printf("どの退会ユーザに残っているかの内訳は -verbose で表示できます\n")
	}
}

// reportSection は1分類分の検出結果を、テーブル単位のサマリとして出力する。
// verbose の場合は、テーブルごとに退会ユーザ単位の内訳も出力する。
func reportSection(findings []finding, names map[string]string, verbose bool) {
	for _, s := range summarizeByTable(findings) {
		log.Printf("  %-26s %6d 件 / 退会ユーザ %d 人 (%s)\n", s.table, s.count, s.userCount, s.note)

		if verbose {
			for _, f := range findings {
				if f.table == s.table {
					log.Printf("      user_id=%s (%s) %d 件\n", f.userId, displayName(names, f.userId), f.count)
				}
			}
		}
	}
}
