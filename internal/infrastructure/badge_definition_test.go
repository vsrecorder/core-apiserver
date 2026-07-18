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

var badgeDefinitionColumns = []string{
	"id", "code", "category", "name", "description", "icon_key",
	"criteria_type", "criteria_value", "available_from", "available_to", "created_at", "updated_at",
}

func TestBadgeDefinitionInfrastructure(t *testing.T) {
	t.Run("正常系_作成日時の昇順で全バッジ定義を返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewBadgeDefinition(db)

		now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.Local)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "badge_definitions" ORDER BY created_at ASC`,
		)).WillReturnRows(sqlmock.NewRows(badgeDefinitionColumns).AddRow(
			"badge-first-record", "first_record", "onboarding", "はじめての記録", "初めて記録を作成した", "icon_first_record",
			"record_count", 1, now, time.Time{}, now, now,
		))

		ret, err := r.FindAll(context.Background())

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, "badge-first-record", ret[0].ID)
		require.Equal(t, "first_record", ret[0].Code)
		require.Equal(t, "onboarding", ret[0].Category)
		require.Equal(t, "record_count", ret[0].CriteriaType)
		require.Equal(t, 1, ret[0].CriteriaValue)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_取得エラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewBadgeDefinition(db)

		mock.ExpectQuery(`SELECT \* FROM "badge_definitions"`).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindAll(context.Background())

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
