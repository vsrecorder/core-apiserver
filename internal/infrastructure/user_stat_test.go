package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUserStatInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	fromDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)

	// FindUserStatは対戦成績→記録数→公式/Tonamel/自由形式イベント数の順に5つのクエリを発行する
	expectStatQueries := func(mock sqlmock.Sqlmock, totalMatches, wins int, records, official, tonamel, unofficial int) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) AS total_matches, SUM\(CASE WHEN matches\.victory_flg = true THEN 1 ELSE 0 END\) AS wins FROM "matches"`).
			WithArgs(uid, fromDate, toDate).
			WillReturnRows(sqlmock.NewRows([]string{"total_matches", "wins"}).AddRow(totalMatches, wins))

		mock.ExpectQuery(`SELECT count\(\*\) FROM "records" WHERE \(user_id = \$1 AND deleted_at IS NULL AND ignore_stats_flg = false\) AND event_date`).
			WithArgs(uid, fromDate, toDate).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(records))

		mock.ExpectQuery(`SELECT COUNT\(DISTINCT\("official_event_id"\)\) FROM "records"`).
			WithArgs(uid, fromDate, toDate).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(official))

		mock.ExpectQuery(`SELECT COUNT\(DISTINCT\("tonamel_event_id"\)\) FROM "records"`).
			WithArgs(uid, fromDate, toDate).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tonamel))

		mock.ExpectQuery(`SELECT COUNT\(DISTINCT\("unofficial_event_id"\)\) FROM "records"`).
			WithArgs(uid, fromDate, toDate).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(unofficial))
	}

	t.Run("正常系_各集計値と勝率を組み立てて返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewUserStat(db)

		expectStatQueries(mock, 10, 6, 5, 2, 1, 1)

		ret, err := r.FindUserStat(context.Background(), uid, fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, uid, ret.UserId)
		require.Equal(t, 5, ret.TotalRecords)
		require.Equal(t, 2, ret.OfficialEventCount)
		require.Equal(t, 1, ret.TonamelEventCount)
		require.Equal(t, 1, ret.UnofficialEventCount)
		require.Equal(t, 10, ret.TotalMatches)
		require.Equal(t, 6, ret.Wins)
		require.Equal(t, 4, ret.Losses)
		require.InDelta(t, 0.6, ret.WinRate, 1e-9)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_対戦が0件なら勝率は0になる", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewUserStat(db)

		expectStatQueries(mock, 0, 0, 0, 0, 0, 0)

		ret, err := r.FindUserStat(context.Background(), uid, fromDate, toDate)

		require.NoError(t, err)
		require.Equal(t, 0, ret.TotalMatches)
		require.Zero(t, ret.WinRate)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_集計クエリのエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewUserStat(db)

		mock.ExpectQuery(`SELECT COUNT\(\*\) AS total_matches`).WillReturnError(sql.ErrConnDone)

		ret, err := r.FindUserStat(context.Background(), uid, fromDate, toDate)

		require.Error(t, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
