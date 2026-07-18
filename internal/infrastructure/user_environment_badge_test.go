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

var userEnvironmentBadgeColumns = []string{
	"user_id", "environment_id", "record_id", "notification_id", "achieved_at", "created_at",
}

func TestUserEnvironmentBadgeInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	achievedAt := time.Date(2026, 7, 18, 11, 0, 0, 0, time.Local)
	createdAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.Local)

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_獲得日時の昇順で指定ユーザの環境バッジを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserEnvironmentBadge(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "user_environment_badges" WHERE user_id = $1 ORDER BY achieved_at ASC`,
			)).WithArgs(uid).WillReturnRows(
				sqlmock.NewRows(userEnvironmentBadgeColumns).AddRow(
					uid, "sv11", "01HD7Y3K8D6FDHMHTZ2GT41TR1", "01HD7Y3K8D6FDHMHTZ2GT41TN9", achievedAt, createdAt,
				),
			)

			ret, err := r.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, uid, ret[0].UserId)
			require.Equal(t, "sv11", ret[0].EnvironmentId)
			require.Equal(t, achievedAt, ret[0].AchievedAt)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("正常系_同一ユーザ同一環境の重複は獲得日時を上書き保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserEnvironmentBadge(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_environment_badges" .*ON CONFLICT \("user_id","environment_id"\) DO UPDATE SET`).
				WithArgs(uid, "sv11", "01HD7Y3K8D6FDHMHTZ2GT41TR1", "01HD7Y3K8D6FDHMHTZ2GT41TN9", achievedAt, createdAt).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			badge := entity.NewUserEnvironmentBadge(uid, "sv11", "01HD7Y3K8D6FDHMHTZ2GT41TR1", "01HD7Y3K8D6FDHMHTZ2GT41TN9", achievedAt, createdAt)

			require.NoError(t, r.Save(context.Background(), badge))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
