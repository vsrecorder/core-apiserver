// list-deleted-users は、退会したユーザ(users.deleted_at IS NOT NULL)の情報を一覧表示する
// 読み取り専用の調査ツール。
//
// 退会処理(internal/usecase/user.go の User.Delete)は、ユーザに紐づくデータをすべて削除するが、
// users の行自体は論理削除で残す。そのため「いつ登録して、いつ・どれだけ使って退会したか」は
// 退会後も users テーブルだけから追える。本ツールはその情報を取り出す。
//
// 表示するのは users テーブルが持つ情報のみで、退会前の活動量(記録・対戦の件数など)は
// 集計しない。退会したユーザが作成したデータが残っていないかの検算は
// cmd/check-deleted-users-data、Firebase Authentication 側との突合は
// cmd/check-firebase-users が担当する。
//
// 判定基準:
//
//   - 退会済み … users.deleted_at IS NOT NULL。gorm.DeletedAt による論理削除は既定の
//     クエリから除外されるため、Unscoped で取得したうえで deleted_at IS NOT NULL に絞る。
//
//   - 利用日数 … 登録(created_at)から退会(deleted_at)までの経過を24時間で割った切り捨て値。
//     データ不整合で退会日が登録日より前になっていても負の値は出さず 0 とする。
//
//   - 期間の絞り込み … -since はその日の0時を含み、-until はその日の翌日0時を含まない
//     半開区間。日付の境界は実行環境のローカルタイムゾーン(本番・開発機はJST)で解釈する。
//
// 本ツールは読み取り専用で、DB に一切書き込みを行わない。何度実行しても結果は変わらない。
// 差異を検出する性質のツールではないため、他の調査ツールが持つ -exit-code は用意していない
// (退会ユーザが存在すること自体は異常ではなく、終了コードで知らせる意味がないため)。
//
// 使い方:
//
//	# 退会ユーザを退会日の新しい順に全件表示する
//	go run ./cmd/list-deleted-users
//
//	# 特定のユーザだけを表示する
//	go run ./cmd/list-deleted-users -user-id <user_id>
//
//	# 退会日で期間を絞る(-since はその日を含み、-until もその日を含む)
//	go run ./cmd/list-deleted-users -since 2026-01-01 -until 2026-03-31
//
//	# 直近の10件だけを表示する
//	go run ./cmd/list-deleted-users -limit 10
//
// 終了コード: 0 = 正常終了、1 = エラー(引数不正・DB接続失敗など)
package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// dateLayout は -since / -until が受け取る日付の書式。
const dateLayout = "2006-01-02"

// deletedUser は表示に使う退会ユーザの情報。
type deletedUser struct {
	ID        string
	Name      string
	CreatedAt time.Time
	DeletedAt time.Time
}

// userCounts は users テーブル全体の内訳。退会ユーザの件数だけを見ても多いのか少ないのか
// 判断できないため、有効なユーザ数と並べて表示するために取得する。
type userCounts struct {
	Total   int64
	Active  int64
	Deleted int64
}

// filterCondition は -user-id / -since / -until による絞り込み条件。
type filterCondition struct {
	// UserID は空文字なら絞り込まない。
	UserID string

	// Since は下限(この時刻を含む)。ゼロ値なら下限なし。
	Since time.Time

	// Until は上限(この時刻を含まない)。ゼロ値なら上限なし。
	// -until に指定された日の翌日0時が入る(その日を含めるため)。
	Until time.Time
}

func main() {
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら退会ユーザ全件)")
	since := flag.String("since", "", "退会日がこの日以降のユーザーだけを対象にする(YYYY-MM-DD。その日を含む)")
	until := flag.String("until", "", "退会日がこの日以前のユーザーだけを対象にする(YYYY-MM-DD。その日を含む)")
	limit := flag.Int("limit", 0, "表示する最大件数(退会日の新しい順。0 で全件)")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	cond, err := buildFilterCondition(*targetUserId, *since, *until, time.Local)
	if err != nil {
		log.Printf("invalid flag: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	if *limit < 0 {
		log.Printf("invalid flag: -limit には0以上を指定してください\n")
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

	counts, err := countUsers(db)
	if err != nil {
		log.Printf("failed to count users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	users, err := listDeletedUsers(db)
	if err != nil {
		log.Printf("failed to list deleted users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	matched := filterUsers(users, cond)

	report(counts, cond, matched, *limit)

	os.Exit(ExitCodeOK)
}

// buildFilterCondition はフラグの値を絞り込み条件へ変換する。
//
// 期間は from の0時 〜 to の翌日0時(含まない)の半開区間として扱う(統計APIの期間の扱いに揃えている)。
// 日付の境界は環境によって変わらないよう loc を明示して解釈する(本番・開発機はJST)。
func buildFilterCondition(userId string, since string, until string, loc *time.Location) (*filterCondition, error) {
	cond := &filterCondition{UserID: userId}

	if since != "" {
		t, err := time.ParseInLocation(dateLayout, since, loc)
		if err != nil {
			return nil, err
		}

		cond.Since = t
	}

	if until != "" {
		t, err := time.ParseInLocation(dateLayout, until, loc)
		if err != nil {
			return nil, err
		}

		// 指定した日そのものを含めるため、翌日0時を「含まない上限」にする
		cond.Until = t.AddDate(0, 0, 1)
	}

	return cond, nil
}

// match は退会ユーザが絞り込み条件に合致するかを返す。
func (c *filterCondition) match(u *deletedUser) bool {
	if c.UserID != "" && u.ID != c.UserID {
		return false
	}

	if !c.Since.IsZero() && u.DeletedAt.Before(c.Since) {
		return false
	}

	if !c.Until.IsZero() && !u.DeletedAt.Before(c.Until) {
		return false
	}

	return true
}

// String は表示用に絞り込み条件を文字列化する。条件が無ければ空文字を返す。
func (c *filterCondition) String() string {
	var conditions []string

	if c.UserID != "" {
		conditions = append(conditions, "user-id="+c.UserID)
	}

	if !c.Since.IsZero() {
		conditions = append(conditions, "since="+c.Since.Format(dateLayout))
	}

	if !c.Until.IsZero() {
		// 内部では「含まない上限」として翌日0時を持っているため、表示は指定された日に戻す
		conditions = append(conditions, "until="+c.Until.AddDate(0, 0, -1).Format(dateLayout))
	}

	return strings.Join(conditions, " ")
}

// countUsers は users テーブルの件数を有効/退会済みに分けて数える。
// 論理削除された行は既定のクエリから除外されるため Unscoped で数える。
func countUsers(db *gorm.DB) (*userCounts, error) {
	var total int64
	if tx := db.Unscoped().Model(&model.User{}).Count(&total); tx.Error != nil {
		return nil, tx.Error
	}

	var deleted int64
	if tx := db.Unscoped().Model(&model.User{}).Where("deleted_at IS NOT NULL").Count(&deleted); tx.Error != nil {
		return nil, tx.Error
	}

	return &userCounts{
		Total:   total,
		Active:  total - deleted,
		Deleted: deleted,
	}, nil
}

// listDeletedUsers は退会したユーザを退会日の新しい順に取得する。
// 「最近だれが退会したか」を最初に見たい場面が多いため降順にし、同じ時刻の場合は
// 実行のたびに並びが変わらないよう id で解決する。
func listDeletedUsers(db *gorm.DB) ([]*deletedUser, error) {
	var rows []*model.User
	if tx := db.Unscoped().Model(&model.User{}).
		Where("deleted_at IS NOT NULL").
		Order("deleted_at DESC, id ASC").
		Find(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	users := make([]*deletedUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, &deletedUser{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			DeletedAt: row.DeletedAt.Time,
		})
	}

	return users, nil
}

// filterUsers は絞り込み条件に合致する退会ユーザだけを、元の並び順を保ったまま返す。
func filterUsers(users []*deletedUser, cond *filterCondition) []*deletedUser {
	matched := make([]*deletedUser, 0, len(users))
	for _, u := range users {
		if cond.match(u) {
			matched = append(matched, u)
		}
	}

	return matched
}

// usageDays は登録から退会までの利用日数(24時間を1日として切り捨て)。
// 時刻の差分で求めるため、DBから読んだ時刻のLocationに依存しない。
// 退会日が登録日より前になっているデータ不整合があっても、負の日数は出さず0とする。
func usageDays(createdAt time.Time, deletedAt time.Time) int {
	duration := deletedAt.Sub(createdAt)
	if duration < 0 {
		return 0
	}

	return int(duration.Hours() / 24)
}

// displayName は表示用のユーザー名。users.name はNULL許容で、退会時に空のまま残ることが
// あるため、空文字が値なのか欠落なのか読み手が迷わないよう明示する。
func displayName(name string) string {
	if name == "" {
		return "(未設定)"
	}

	return name
}

// report は結果を標準出力へ出力する。limit が 0 より大きい場合は先頭 limit 件だけを表示する。
func report(counts *userCounts, cond *filterCondition, users []*deletedUser, limit int) {
	log.Printf("users: 全 %d 件 (有効: %d, 退会済み: %d)\n", counts.Total, counts.Active, counts.Deleted)

	if condition := cond.String(); condition != "" {
		log.Printf("条件: %s\n", condition)
	}

	if len(users) == 0 {
		log.Printf("対象: 0 件 (条件に合致する退会ユーザーはありません)\n")
		return
	}

	shown := users
	if limit > 0 && limit < len(shown) {
		shown = shown[:limit]
		log.Printf("対象: %d 件 (退会日の新しい順に %d 件のみ表示)\n", len(users), len(shown))
	} else {
		log.Printf("対象: %d 件\n", len(users))
	}

	for _, u := range shown {
		log.Printf("  uid=%s name=%s created_at=%s deleted_at=%s 利用日数=%d\n",
			u.ID,
			displayName(u.Name),
			u.CreatedAt.Format(time.RFC3339),
			u.DeletedAt.Format(time.RFC3339),
			usageDays(u.CreatedAt, u.DeletedAt),
		)
	}
}
