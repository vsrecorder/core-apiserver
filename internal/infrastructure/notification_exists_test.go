package infrastructure

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestNotificationInfrastructure_ExistsByUserIdAndCategoryAndLinkUrl(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_一致する通知があればtrue", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewNotification(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "notifications" WHERE user_id = \$1 AND category = \$2 AND link_url = \$3`).
			WithArgs(uid, "weekly_report", "/users/report/weeks/2026-08-17").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		got, err := r.ExistsByUserIdAndCategoryAndLinkUrl(context.Background(), uid, "weekly_report", "/users/report/weeks/2026-08-17")

		require.NoError(t, err)
		require.True(t, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_無ければfalse", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewNotification(db)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "notifications"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		got, err := r.ExistsByUserIdAndCategoryAndLinkUrl(context.Background(), uid, "weekly_report", "/users/report/weeks/2026-08-10")

		require.NoError(t, err)
		require.False(t, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
