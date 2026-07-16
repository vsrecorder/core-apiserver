// check-deleted-users-data は、退会したユーザ(users.deleted_at IS NOT NULL)が作成した
// データが残っていないかを確認するための調査用ツール。
//
// 退会処理(internal/usecase/user.go の User.Delete)は、ユーザ本体を論理削除する前に
// 対戦記録・デッキ・デッキコード・プレイヤーID紐付けを連鎖削除している。本ツールは
// その結果を DB 側から検算し、消えているはずのデータが残っていないかを検出する。
//
// 検出結果は、原因と取るべき対処が異なるため次の3つに分類して表示する:
//
//   - 削除漏れ(NG)    … 退会処理が削除対象としているのに残っているデータ。退会処理が途中で
//     失敗した、もしくは退会処理の実装後に作られた経路(バッチ等)で
//     データが作られた可能性がある。調査と手動での削除が必要。
//
//   - 未対応(WARN)    … そもそも退会処理が削除対象にしていないテーブルに残っているデータ。
//     バグではなく仕様上の未対応であり、対処するなら退会処理の実装を
//     変更する必要がある。実行するたびに必ず検出される点に注意。
//
//   - 参照(INFO)      … 他のユーザが作成したデータから、退会したユーザが参照されているもの。
//     他人のデータなので消してはいけない。異常ではない。
//
// 「残っている」の判定は、論理削除(gorm.DeletedAt)を持つテーブルは deleted_at IS NULL の
// 行、持たないテーブルは行の存在そのもの、としている。
//
// 本ツールは読み取り専用で、DB に一切書き込みを行わない。
//
// 使い方:
//
//	# 退会ユーザ全体のサマリを表示する
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
// -exit-code が終了コード1にするのは「削除漏れ(NG)」があった場合のみで、「未対応(WARN)」は
// 対象にしない。未対応は退会処理の仕様上つねに残るため、これを異常扱いにすると定期実行が
// 鳴りっぱなしになるため。
//
// 終了コード: 0 = 正常終了(-exit-code 指定時は削除漏れなし)、1 = エラー(または削除漏れあり)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

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

// tableSpec は「退会したユーザのデータが残っていないか」を1テーブル分確認するための定義。
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
		note: "User.Delete が user_id で洗い出して削除する",
	},
	{
		name:     "matches",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM matches t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		note: "records の削除に連鎖して削除される。record を経由しない孤立行があると残る",
	},
	{
		name:     "games",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM games t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		note: "matches の削除に連鎖して削除される。match を経由しない孤立行があると残る",
	},
	{
		name:     "decks",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM decks t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		note: "User.Delete が user_id で洗い出して削除する",
	},
	{
		name:     "deck_codes",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM deck_codes t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		note: "User.Delete が user_id で洗い出して削除する",
	},
	{
		name:     "users_players",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM users_players t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		note: "User.Delete が FindByUserId で引いた1件を削除する。有効な行が2件以上あると残る",
	},
	{
		name:     "unofficial_events",
		category: categoryLeak,
		query: `SELECT t.user_id, COUNT(*) FROM unofficial_events t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL AND t.deleted_at IS NULL
		        GROUP BY t.user_id`,
		// Record.Delete は records.unofficial_event_id をたどって削除するため、どの record からも
		// 参照されていない自由形式イベントは退会処理では消えない。残った場合はこの可能性が高い。
		note: "record 経由でのみ削除される。どの record からも参照されていないものは残る",
	},

	// --- 退会処理が削除対象にしていないもの(残っていても実装どおり) ---
	{
		name:     "user_streaks",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM user_streaks t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		note: "退会処理の対象外。論理削除を持たないため行ごと残る",
	},
	{
		name:     "user_badges",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM user_badges t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		note: "退会処理の対象外。論理削除を持たないため行ごと残る",
	},
	{
		name:     "user_environment_badges",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM user_environment_badges t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		note: "退会処理の対象外。論理削除を持たないため行ごと残る",
	},
	{
		name:     "notifications",
		category: categoryUnhandled,
		query: `SELECT t.user_id, COUNT(*) FROM notifications t
		        JOIN users u ON u.id = t.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY t.user_id`,
		note: "退会処理の対象外。論理削除を持たないため行ごと残る",
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
		note: "退会処理の対象外。match が論理削除されてもスプライトの行は残る",
	},
	{
		name:     "deck_pokemon_sprites",
		category: categoryUnhandled,
		query: `SELECT d.user_id, COUNT(*) FROM deck_pokemon_sprites t
		        JOIN decks d ON d.id = t.deck_id
		        JOIN users u ON u.id = d.user_id
		        WHERE u.deleted_at IS NOT NULL
		        GROUP BY d.user_id`,
		note: "退会処理の対象外。deck が論理削除されてもスプライトの行は残る",
	},

	// --- 他のユーザのデータからの参照(異常ではない) ---
	{
		name:     "matches.opponents_user_id",
		category: categoryReference,
		// 退会したユーザ「が」作成したデータではなく、他のユーザが作成した対戦記録の中で
		// 対戦相手として参照されているもの。他人のデータなので退会処理では消さないのが正しい。
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

func main() {
	verbose := flag.Bool("verbose", false, "どの退会ユーザにどれだけ残っているかの内訳を表示する")
	userId := flag.String("user", "", "確認対象の退会ユーザのIDを1件に絞る(未指定なら退会ユーザ全件)")
	exitCode := flag.Bool("exit-code", false, "true の場合、削除漏れが見つかったら終了コード1で終了する")
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

	deletedUserCount, err := countDeletedUsers(db, *userId)
	if err != nil {
		log.Printf("failed to count deleted users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	if deletedUserCount == 0 {
		if *userId != "" {
			log.Printf("確認対象の退会ユーザが見つかりません(user_id=%s は存在しないか、まだ退会していません)\n", *userId)
			os.Exit(ExitCodeNG)
		}

		log.Printf("退会したユーザ(deleted_at IS NOT NULL)が1人もいないため、確認対象がありません\n")
		os.Exit(ExitCodeOK)
	}

	if *userId != "" {
		log.Printf("退会ユーザ user_id=%s のデータを確認します\n", *userId)
	} else {
		log.Printf("退会ユーザ %d 人のデータを確認します\n", deletedUserCount)
	}

	findings, err := collect(db, specs, *userId)
	if err != nil {
		log.Printf("failed to collect findings: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	report(findings, *verbose)

	if *exitCode && len(filterByCategory(findings, categoryLeak)) > 0 {
		os.Exit(ExitCodeNG)
	}

	os.Exit(ExitCodeOK)
}

// countDeletedUsers は確認対象の退会ユーザ数を返す。userId を指定した場合は、そのユーザが
// 退会済みとして存在すれば1、そうでなければ0を返す。
func countDeletedUsers(db *gorm.DB, userId string) (int64, error) {
	query := db.Table("users").Where("deleted_at IS NOT NULL")
	if userId != "" {
		query = query.Where("id = ?", userId)
	}

	var count int64
	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return count, nil
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

// report は確認結果を標準出力へ出力する。
func report(findings []finding, verbose bool) {
	leaks := filterByCategory(findings, categoryLeak)
	unhandled := filterByCategory(findings, categoryUnhandled)
	references := filterByCategory(findings, categoryReference)

	if len(leaks) == 0 {
		log.Printf("OK: 退会処理が削除対象としているデータは、すべて削除されています\n")
	} else {
		log.Printf("NG: 削除漏れ(退会処理が削除するはずのデータが残っています)\n")
		reportSection(leaks, verbose)
	}

	if len(unhandled) > 0 {
		log.Printf("WARN: 未対応(退会処理が削除対象にしていないデータが残っています)\n")
		log.Printf("      バグではなく仕様上の未対応です。消すなら退会処理(internal/usecase/user.go)の実装変更が必要です\n")
		reportSection(unhandled, verbose)
	}

	if len(references) > 0 {
		log.Printf("INFO: 参照(他のユーザが作成したデータから、退会したユーザが参照されています)\n")
		log.Printf("      他人のデータのため退会処理では削除しません。異常ではありません\n")
		reportSection(references, verbose)
	}

	if !verbose && len(findings) > 0 {
		log.Printf("どの退会ユーザに残っているかの内訳は -verbose で表示できます\n")
	}
}

// reportSection は1分類分の検出結果を、テーブル単位のサマリとして出力する。
// verbose の場合は、テーブルごとに退会ユーザ単位の内訳も出力する。
func reportSection(findings []finding, verbose bool) {
	for _, s := range summarizeByTable(findings) {
		log.Printf("  %-26s %6d 件 / 退会ユーザ %d 人 (%s)\n", s.table, s.count, s.userCount, s.note)

		if verbose {
			for _, f := range findings {
				if f.table == s.table {
					log.Printf("      user_id=%s %d 件\n", f.userId, f.count)
				}
			}
		}
	}
}
