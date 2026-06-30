package infrastructure

import (
	"database/sql/driver"
	"errors"
	"math/rand"
	"time"

	ulid "github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
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
