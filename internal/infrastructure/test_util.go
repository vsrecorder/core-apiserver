package infrastructure

import (
	"database/sql/driver"
	"time"
)

// GORM側で更新される カラム updated_at, deleted_at の値をテストでPASSするための構造体
type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}
