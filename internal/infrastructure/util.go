package infrastructure

import (
	"database/sql/driver"
	"math/rand"
	"time"

	ulid "github.com/oklog/ulid/v2"
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
