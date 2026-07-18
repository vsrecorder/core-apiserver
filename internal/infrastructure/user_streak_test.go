package infrastructure

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

var userStreakColumns = []string{
	"user_id", "current_weeks", "longest_weeks", "freeze_used_count", "last_recorded_week", "updated_at",
}

func TestUserStreakInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	lastRecordedWeek := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
	updatedAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.Local)

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_指定ユーザのストリークを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserStreak(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "user_streaks" WHERE user_id = $1 ORDER BY "user_streaks"."user_id" LIMIT $2`,
			)).WithArgs(uid, 1).WillReturnRows(
				sqlmock.NewRows(userStreakColumns).AddRow(uid, 3, 5, 1, lastRecordedWeek, updatedAt),
			)

			ret, err := r.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, 3, ret.CurrentWeeks)
			require.Equal(t, 5, ret.LongestWeeks)
			require.Equal(t, 1, ret.FreezeUsedCount)
			require.Equal(t, lastRecordedWeek, ret.LastRecordedWeek)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_該当なしはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserStreak(db)

			mock.ExpectQuery(`SELECT \* FROM "user_streaks"`).
				WithArgs(uid, 1).WillReturnRows(sqlmock.NewRows(userStreakColumns))

			ret, err := r.FindByUserId(context.Background(), uid)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("正常系_ストリークを全項目更新で保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserStreak(db)

			mock.ExpectBegin()
			// updated_atはGORMが保存時刻で上書きするためAnyTimeで検証する
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "user_streaks" SET "current_weeks"=$1,"longest_weeks"=$2,"freeze_used_count"=$3,"last_recorded_week"=$4,"updated_at"=$5 WHERE "user_id" = $6`,
			)).WithArgs(
				4, 5, 1, lastRecordedWeek, AnyTime{}, uid,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			streak := entity.NewUserStreak(uid, 4, 5, 1, lastRecordedWeek, updatedAt)

			require.NoError(t, r.Save(context.Background(), streak))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
