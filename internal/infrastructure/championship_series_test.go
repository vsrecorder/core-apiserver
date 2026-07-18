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

var championshipSeriesColumns = []string{"id", "title", "from_date", "to_date"}

func TestChampionshipSeriesInfrastructure(t *testing.T) {
	fromDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_開始日の降順で全シリーズを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionshipSeries(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championship_series" ORDER BY from_date DESC`,
			)).WillReturnRows(sqlmock.NewRows(championshipSeriesColumns).AddRow(
				"series_2026", "チャンピオンシップシリーズ2026", fromDate, toDate,
			))

			ret, err := r.Find(context.Background())

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, "series_2026", ret[0].ID)
			require.Equal(t, "チャンピオンシップシリーズ2026", ret[0].Title)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDのシリーズを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionshipSeries(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championship_series" WHERE id = $1 ORDER BY "championship_series"."id" LIMIT $2`,
			)).WithArgs("series_2026", 1).WillReturnRows(
				sqlmock.NewRows(championshipSeriesColumns).AddRow("series_2026", "チャンピオンシップシリーズ2026", fromDate, toDate),
			)

			ret, err := r.FindById(context.Background(), "series_2026")

			require.NoError(t, err)
			require.Equal(t, "series_2026", ret.ID)
			require.Equal(t, fromDate, ret.FromDate)
			require.Equal(t, toDate, ret.ToDate)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionshipSeries(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championship_series" WHERE id = $1`,
			)).WithArgs("series_9999", 1).WillReturnRows(sqlmock.NewRows(championshipSeriesColumns))

			ret, err := r.FindById(context.Background(), "series_9999")

			require.Equal(t, apperror.ErrRecordNotFound, err)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindByDate", func(t *testing.T) {
		t.Run("正常系_指定日を期間に含むシリーズを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionshipSeries(db)

			date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championship_series" WHERE from_date <= $1 AND to_date >= $2 ORDER BY "championship_series"."id" LIMIT $3`,
			)).WithArgs(date, date, 1).WillReturnRows(
				sqlmock.NewRows(championshipSeriesColumns).AddRow("series_2026", "チャンピオンシップシリーズ2026", fromDate, toDate),
			)

			ret, err := r.FindByDate(context.Background(), date)

			require.NoError(t, err)
			require.Equal(t, "series_2026", ret.ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_該当なしはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionshipSeries(db)

			date := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championship_series" WHERE from_date <= $1 AND to_date >= $2`,
			)).WithArgs(date, date, 1).WillReturnRows(sqlmock.NewRows(championshipSeriesColumns))

			ret, err := r.FindByDate(context.Background(), date)

			require.Equal(t, apperror.ErrRecordNotFound, err)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
