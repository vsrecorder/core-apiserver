package infrastructure

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

var designationColumns = []string{
	"id", "tier", "code", "emoji", "name", "description",
	"criteria_type", "criteria_value", "created_at", "updated_at",
}

func TestDesignationInfrastructure(t *testing.T) {
	t.Run("正常系_tierの昇順で全称号定義を返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewDesignation(db)

		now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.Local)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "designations" ORDER BY tier ASC`,
		)).WillReturnRows(sqlmock.NewRows(designationColumns).AddRow(
			"designation-rookie", 1, "rookie", "🔰", "ルーキー", "記録を1件作成", "record_count", 1, now, now,
		).AddRow(
			"designation-veteran", 5, "veteran", "🎖", "ベテラン", "シティリーグ出場", "cityleague", 1, now, now,
		))

		ret, err := r.FindAll(context.Background())

		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, "designation-rookie", ret[0].ID)
		require.Equal(t, 1, ret[0].Tier)
		require.Equal(t, "rookie", ret[0].Code)
		require.Equal(t, "designation-veteran", ret[1].ID)
		require.Equal(t, 5, ret[1].Tier)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_取得エラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewDesignation(db)

		mock.ExpectQuery(`SELECT \* FROM "designations"`).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindAll(context.Background())

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
