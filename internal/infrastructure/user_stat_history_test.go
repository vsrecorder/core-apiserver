package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUserStatHistoryInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	fromDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)

	monthlyColumns := []string{"year_month", "total_matches", "wins"}

	t.Run("正常系_月ごとの対戦数と勝率を組み立てて返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewUserStatHistory(db)

		mock.ExpectQuery(`SELECT TO_CHAR\(DATE_TRUNC\('month', records\.event_date\), 'YYYY-MM'\) AS year_month`).
			WithArgs(uid, fromDate, toDate).
			WillReturnRows(sqlmock.NewRows(monthlyColumns).
				AddRow("2026-05", 4, 3).
				AddRow("2026-06", 5, 0),
			)

		ret, err := r.FindUserStatHistory(context.Background(), uid, fromDate, toDate, "")

		require.NoError(t, err)
		require.Len(t, ret, 2)

		require.Equal(t, "2026-05", ret[0].YearMonth)
		require.Equal(t, 4, ret[0].TotalMatches)
		require.Equal(t, 3, ret[0].Wins)
		require.Equal(t, 1, ret[0].Losses)
		require.InDelta(t, 0.75, ret[0].WinRate, 1e-9)

		require.Equal(t, "2026-06", ret[1].YearMonth)
		require.Zero(t, ret[1].WinRate)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_deck_id指定時は絞り込み条件に加える", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewUserStatHistory(db)

		deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		mock.ExpectQuery(`SELECT TO_CHAR.*matches\.deck_id = \$4.*GROUP BY`).
			WithArgs(uid, fromDate, toDate, deckId).
			WillReturnRows(sqlmock.NewRows(monthlyColumns))

		ret, err := r.FindUserStatHistory(context.Background(), uid, fromDate, toDate, deckId)

		require.NoError(t, err)
		require.Empty(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_集計クエリのエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewUserStatHistory(db)

		mock.ExpectQuery(`SELECT TO_CHAR`).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindUserStatHistory(context.Background(), uid, fromDate, toDate, "")

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
