package infrastructure

import (
	"context"
	"database/sql/driver"
	"errors"
	"log/slog"
	"math/rand"
	"time"

	ulid "github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

var (
	entropy    = rand.New(rand.NewSource(time.Now().UnixNano()))
	DateLayout = time.DateOnly
)

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
