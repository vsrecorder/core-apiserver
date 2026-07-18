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

var userBadgeColumns = []string{
	"id", "created_at", "user_id", "badge_definition_id", "record_id", "achieved_at",
}

func TestUserBadgeInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	createdAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.Local)
	achievedAt := time.Date(2026, 7, 18, 11, 0, 0, 0, time.Local)

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_獲得日時の昇順で指定ユーザのバッジを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserBadge(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "user_badges" WHERE user_id = $1 ORDER BY achieved_at ASC`,
			)).WithArgs(uid).WillReturnRows(
				sqlmock.NewRows(userBadgeColumns).AddRow(
					id, createdAt, uid, "badge-first-record", "01HD7Y3K8D6FDHMHTZ2GT41TR1", achievedAt,
				),
			)

			ret, err := r.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, id, ret[0].ID)
			require.Equal(t, "badge-first-record", ret[0].BadgeDefinitionId)
			require.Equal(t, achievedAt, ret[0].AchievedAt)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("正常系_同一ユーザ同一バッジの重複はDoNothingで冪等に保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserBadge(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_badges" .*ON CONFLICT \("user_id","badge_definition_id"\) DO NOTHING`).
				WithArgs(id, createdAt, uid, "badge-first-record", "01HD7Y3K8D6FDHMHTZ2GT41TR1", achievedAt).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			badge := entity.NewUserBadge(id, createdAt, uid, "badge-first-record", "01HD7Y3K8D6FDHMHTZ2GT41TR1", achievedAt)

			require.NoError(t, r.Save(context.Background(), badge))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
