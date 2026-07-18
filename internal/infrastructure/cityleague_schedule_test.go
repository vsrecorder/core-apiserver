package infrastructure

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

var cityleagueScheduleColumns = []string{"id", "title", "from_date", "to_date"}

func TestCityleagueScheduleInfrastructure(t *testing.T) {
	fromDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_開始日の降順で全日程を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewCityleagueSchedule(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "cityleague_schedules" ORDER BY from_date DESC`,
			)).WillReturnRows(sqlmock.NewRows(cityleagueScheduleColumns).AddRow(
				"2026_s1", "シティリーグ2026 シーズン1", fromDate, toDate,
			))

			ret, err := r.Find(context.Background())

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, "2026_s1", ret[0].ID)
			require.Equal(t, "シティリーグ2026 シーズン1", ret[0].Title)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDの日程を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewCityleagueSchedule(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "cityleague_schedules" WHERE id = $1 ORDER BY "cityleague_schedules"."id" LIMIT $2`,
			)).WithArgs("2026_s1", 1).WillReturnRows(
				sqlmock.NewRows(cityleagueScheduleColumns).AddRow("2026_s1", "シティリーグ2026 シーズン1", fromDate, toDate),
			)

			ret, err := r.FindById(context.Background(), "2026_s1")

			require.NoError(t, err)
			require.Equal(t, "2026_s1", ret.ID)
			require.Equal(t, fromDate, ret.FromDate)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewCityleagueSchedule(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "cityleague_schedules" WHERE id = $1`,
			)).WithArgs("unknown", 1).WillReturnRows(sqlmock.NewRows(cityleagueScheduleColumns))

			ret, err := r.FindById(context.Background(), "unknown")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindByDate", func(t *testing.T) {
		t.Run("正常系_指定日を期間に含む日程を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewCityleagueSchedule(db)

			date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "cityleague_schedules" WHERE from_date <= $1 AND to_date >= $2 ORDER BY "cityleague_schedules"."id" LIMIT $3`,
			)).WithArgs(date, date, 1).WillReturnRows(
				sqlmock.NewRows(cityleagueScheduleColumns).AddRow("2026_s1", "シティリーグ2026 シーズン1", fromDate, toDate),
			)

			ret, err := r.FindByDate(context.Background(), date)

			require.NoError(t, err)
			require.Equal(t, "2026_s1", ret.ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_該当なしはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewCityleagueSchedule(db)

			date := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "cityleague_schedules" WHERE from_date <= $1 AND to_date >= $2`,
			)).WithArgs(date, date, 1).WillReturnRows(sqlmock.NewRows(cityleagueScheduleColumns))

			ret, err := r.FindByDate(context.Background(), date)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
