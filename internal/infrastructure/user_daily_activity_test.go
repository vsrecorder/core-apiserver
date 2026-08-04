package infrastructure

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func TestUserDailyActivityInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	date := time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)
	updatedAt := time.Date(2026, 8, 4, 12, 34, 56, 0, time.Local)

	t.Run("Touch", func(t *testing.T) {
		t.Run("正常系_複数カテゴリを1文のupsertで発行する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserDailyActivity(db)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`INSERT INTO "user_daily_activities" ("user_id","date","category","signal_count","updated_at") VALUES ($1,$2,$3,$4,$5),($6,$7,$8,$9,$10) ON CONFLICT ("user_id","date","category") DO UPDATE SET "signal_count"=user_daily_activities.signal_count + excluded.signal_count,"updated_at"=excluded.updated_at`,
			)).WithArgs(
				uid, date, entity.UserDailyActivityCategoryVisit, 1, AnyTime{},
				uid, date, entity.UserDailyActivityCategoryReview, 1, AnyTime{},
			).WillReturnResult(sqlmock.NewResult(0, 2))
			mock.ExpectCommit()

			err := r.Touch(context.Background(), []*entity.UserDailyActivity{
				entity.NewUserDailyActivity(uid, date, entity.UserDailyActivityCategoryVisit, updatedAt),
				entity.NewUserDailyActivity(uid, date, entity.UserDailyActivityCategoryReview, updatedAt),
			})

			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_空スライスならクエリを発行しない", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserDailyActivity(db)

			require.NoError(t, r.Touch(context.Background(), nil))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
