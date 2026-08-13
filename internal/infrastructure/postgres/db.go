package postgres

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 常時稼働するAPIサーバ本体を想定した既定値。
// cmd配下のバッチコマンドも同じNewDBを使うため、それらは環境変数
// DB_MAX_OPEN_CONNS でより小さい値に絞ってPostgreSQL側の枠を空ける。
const defaultMaxOpenConns = 25

const (
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute
)

func NewDB(
	host string,
	port string,
	user string,
	password string,
	dbname string,
) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Tokyo",
		host, port, user, password, dbname,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// GORMは内部でdatabase/sqlを使うが、プールは初期値
	// (MaxOpenConns=無制限 / MaxIdleConns=2 / Lifetime=無期限) のままになる。
	// 無制限だと負荷時にPostgreSQLのmax_connectionsを使い切りOOMに至るため明示する。
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	maxOpenConns := defaultMaxOpenConns
	if v, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS")); err == nil && v > 0 {
		maxOpenConns = v
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	// 既定の2では接続の張り直し(PostgreSQL側のプロセスfork)が頻発するため上限の半分を保持する
	sqlDB.SetMaxIdleConns(max(1, maxOpenConns/2))
	// PostgreSQL再起動後の復帰を速くするため接続を定期的に張り直す
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	return db, nil
}
