package infrastructure

import (
	"context"
	"database/sql/driver"
	"errors"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	ulid "github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

var (
	entropy    = rand.New(rand.NewSource(time.Now().UnixNano()))
	DateLayout = time.DateOnly
)

// localDate は DATE カラムから読み出した値を、ローカル時刻の同じ暦日 0時 に揃えて返す。
//
// GORM(pgx)は DATE を UTC の 0時 として返す一方、TIMESTAMP はローカル時刻で返し、
// time.Now() もローカル時刻のため、同じ暦日でも time.Time としては別の瞬間になる。
// usecase 側で「同じ週か」「何週あいたか」を突き合わせる日付(週次ストリーク関連)は、
// 由来によらずここでローカルの暦日に揃えてから返す。ゼロ値(未入力)はそのまま返す。
func localDate(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// GORM側で更新される カラム updated_at, deleted_at の値をテストでPASSするための構造体
type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}

// wrapError は gorm の永続化エラーをドメインエラーへ変換する。
// レコードが存在しない場合(gorm.ErrRecordNotFound)は apperror.ErrRecordNotFound
// へ変換し、上位層が gorm に依存せずエラーの種類を判定できるようにする。
// それ以外のエラーはそのまま返す。
func wrapError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.ErrRecordNotFound
	}

	return err
}

// wrapUniqueViolation は一意制約違反(主キー重複)を apperror.ErrAlreadyExists へ変換する。
//
// wrapError と分けてあるのは、これを全リポジトリへ一律に適用すると、これまで500だった
// 制約違反が上位で「既に存在する」(409)として扱われ、意図しない応答に変わりうるため。
// 重複が正常な入力として起こりうる箇所(同じ対象への同時登録など)でだけ使う。
func wrapUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	// 23505 = unique_violation
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperror.ErrAlreadyExists
	}

	return err
}

// logError は永続化層(DB・S3・外部サイト)で発生したエラーをログへ出力する。
//
// 呼び出し元のメソッド名とソース位置は runtime から解決するため、呼び出し側は
// logError(ctx, err) と書くだけでよい。request_id / uid は ctx から
// ContextHandler が自動で付与する。
//
// レコードが存在しないことは障害ではなく想定内の結果(未登録ユーザの初回アクセス等)
// であるため Debug へ落とし、Error レベルを実際に調査が必要なものだけに保つ。
func logError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	level := slog.LevelError
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, apperror.ErrRecordNotFound) {
		level = slog.LevelDebug
	}

	logging.LogAt(
		ctx,
		logging.Layered(logging.LayerInfrastructure),
		level,
		1,
		"repository operation failed",
		logging.Operation(logging.CallerOperation(1)),
		logging.Err(err),
	)
}
