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

var championsleagueScheduleColumns = []string{"id", "title", "from_date", "to_date"}

func TestChampionsleagueScheduleInfrastructure(t *testing.T) {
	fromDate := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_開始日の降順で全大会を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionsleagueSchedule(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championsleague_schedules" ORDER BY from_date DESC`,
			)).WillReturnRows(sqlmock.NewRows(championsleagueScheduleColumns).AddRow(
				"cl2027_yokohama", "チャンピオンズリーグ2027 横浜", fromDate, toDate,
			))

			ret, err := r.Find(context.Background())

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, "cl2027_yokohama", ret[0].ID)
			require.Equal(t, "チャンピオンズリーグ2027 横浜", ret[0].Title)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_1件も無い場合は空スライスを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionsleagueSchedule(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championsleague_schedules"`,
			)).WillReturnRows(sqlmock.NewRows(championsleagueScheduleColumns))

			ret, err := r.Find(context.Background())

			require.NoError(t, err)
			require.NotNil(t, ret)
			require.Empty(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDの大会を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionsleagueSchedule(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championsleague_schedules" WHERE id = $1 ORDER BY "championsleague_schedules"."id" LIMIT $2`,
			)).WithArgs("cl2027_yokohama", 1).WillReturnRows(
				sqlmock.NewRows(championsleagueScheduleColumns).AddRow(
					"cl2027_yokohama", "チャンピオンズリーグ2027 横浜", fromDate, toDate,
				),
			)

			ret, err := r.FindById(context.Background(), "cl2027_yokohama")

			require.NoError(t, err)
			require.Equal(t, "cl2027_yokohama", ret.ID)
			require.Equal(t, fromDate, ret.FromDate)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewChampionsleagueSchedule(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "championsleague_schedules" WHERE id = $1`,
			)).WithArgs("unknown", 1).WillReturnRows(sqlmock.NewRows(championsleagueScheduleColumns))

			ret, err := r.FindById(context.Background(), "unknown")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
