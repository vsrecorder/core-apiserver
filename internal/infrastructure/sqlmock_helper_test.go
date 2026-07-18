package infrastructure

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupSqlmockDB はsqlmockを接続に使うgorm.DBを返す共通ヘルパー。
// 各リポジトリのテストはこれで得たDBからリポジトリを組み立てる。
func setupSqlmockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn: mockDB,
		}),
		&gorm.Config{},
	)
	require.NoError(t, err)

	return db, mock
}
