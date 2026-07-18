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

var standardRegulationColumns = []string{"id", "marks", "from_date", "to_date"}

func TestStandardRegulationInfrastructure(t *testing.T) {
	fromDate := time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2027, 1, 22, 0, 0, 0, 0, time.UTC)

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_開始日の降順で全レギュレーションを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewStandardRegulation(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "standard_regulations" ORDER BY from_date DESC`,
			)).WillReturnRows(sqlmock.NewRows(standardRegulationColumns).AddRow(
				"regulation-h", "H,I,J", fromDate, toDate,
			).AddRow(
				"regulation-g", "G,H,I", fromDate.AddDate(-1, 0, 0), toDate.AddDate(-1, 0, 0),
			))

			ret, err := r.Find(context.Background())

			require.NoError(t, err)
			require.Len(t, ret, 2)
			require.Equal(t, "regulation-h", ret[0].ID)
			require.Equal(t, "H,I,J", ret[0].Marks)
			require.Equal(t, "regulation-g", ret[1].ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDのレギュレーションを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewStandardRegulation(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "standard_regulations" WHERE id = $1 ORDER BY "standard_regulations"."id" LIMIT $2`,
			)).WithArgs("regulation-g", 1).WillReturnRows(
				sqlmock.NewRows(standardRegulationColumns).AddRow("regulation-g", "G,H,I", fromDate, toDate),
			)

			ret, err := r.FindById(context.Background(), "regulation-g")

			require.NoError(t, err)
			require.Equal(t, "regulation-g", ret.ID)
			require.Equal(t, "G,H,I", ret.Marks)
			require.Equal(t, fromDate, ret.FromDate)
			require.Equal(t, toDate, ret.ToDate)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewStandardRegulation(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "standard_regulations" WHERE id = $1`,
			)).WithArgs("unknown", 1).WillReturnRows(sqlmock.NewRows(standardRegulationColumns))

			ret, err := r.FindById(context.Background(), "unknown")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindByDate", func(t *testing.T) {
		t.Run("正常系_指定日を期間に含むレギュレーションを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewStandardRegulation(db)

			date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "standard_regulations" WHERE from_date <= $1 AND to_date >= $2 ORDER BY "standard_regulations"."id" LIMIT $3`,
			)).WithArgs(date, date, 1).WillReturnRows(
				sqlmock.NewRows(standardRegulationColumns).AddRow("regulation-g", "G,H,I", fromDate, toDate),
			)

			ret, err := r.FindByDate(context.Background(), date)

			require.NoError(t, err)
			require.Equal(t, "regulation-g", ret.ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_該当なしはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewStandardRegulation(db)

			date := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "standard_regulations" WHERE from_date <= $1 AND to_date >= $2`,
			)).WithArgs(date, date, 1).WillReturnRows(sqlmock.NewRows(standardRegulationColumns))

			ret, err := r.FindByDate(context.Background(), date)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
