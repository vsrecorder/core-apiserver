package main

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// specs のSQLは手書きの文字列で、単体テストでは形(WHERE句や列の有無)しか見ていない。
// 列名やテーブル名の打ち間違いは実行するまで分からず、定期実行のたびに落ちることになる。
// ここで実スキーマに対して全定義を実際に流し、SQLとして通ることを確認する。
func TestIntegrationSpecsSQL(t *testing.T) {
	dsn := os.Getenv("VSRECORDER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VSRECORDER_TEST_DATABASE_URL が未設定のためスキップ(make integration-test で実行できます)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	const uid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 検証用にロールバックさせるための番兵。
	errRollback := errors.New("rollback")

	for _, spec := range specs {
		t.Run("正常系_"+spec.name, func(t *testing.T) {
			// 検出クエリ: 退会ユーザ全件版と -user 指定版の両方を流す。
			// 後者は spec.query をサブクエリに包むため、列の別名まで含めて通る必要がある。
			// 他の統合テストが同じDBを使い回すため、検出結果の中身は当てにしない。
			// ここで見たいのは「SQLとして実行でき、期待した形(2列)で返ること」。
			for _, userId := range []string{"", uid} {
				_, err := queryCounts(db, spec, userId)
				require.NoError(t, err, "検出クエリが実行できない")
			}

			if spec.deleteQuery == "" {
				return
			}

			// 削除クエリ: 実際に消してしまわないよう、必ずロールバックする。
			// -user 指定時に AND で条件を足す形も本体と同じ組み立てで確認する。
			for _, userId := range []string{"", uid} {
				err := db.Transaction(func(tx *gorm.DB) error {
					query := spec.deleteQuery
					var args []any
					if userId != "" {
						query += " AND " + spec.ownerColumn + " = ?"
						args = append(args, userId)
					}

					if err := tx.Exec(query, args...).Error; err != nil {
						return err
					}

					return errRollback
				})
				require.ErrorIs(t, err, errRollback, "削除クエリが実行できない")
			}
		})
	}
}
