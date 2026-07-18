package infrastructure

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

var userPlayerColumns = []string{
	"id", "created_at", "updated_at", "deleted_at", "user_id", "player_id",
}

func TestUserPlayerInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	playerId := "1234567890123456"

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_指定ユーザの紐付けを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserPlayer(db)

			now := time.Now().Local()

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "users_players" WHERE user_id = $1 AND "users_players"."deleted_at" IS NULL ORDER BY "users_players"."id" LIMIT $2`,
			)).WithArgs(uid, 1).WillReturnRows(
				sqlmock.NewRows(userPlayerColumns).AddRow(id, now, now, gorm.DeletedAt{}, uid, playerId),
			)

			ret, err := r.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, playerId, ret.PlayerId)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_紐付けが無ければErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserPlayer(db)

			mock.ExpectQuery(`SELECT \* FROM "users_players"`).
				WithArgs(uid, 1).WillReturnRows(sqlmock.NewRows(userPlayerColumns))

			ret, err := r.FindByUserId(context.Background(), uid)

			require.Equal(t, apperror.ErrRecordNotFound, err)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("ExistsActiveByPlayerId", func(t *testing.T) {
		t.Run("正常系_有効な紐付けがあればtrueを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserPlayer(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT count(*) FROM "users_players" WHERE player_id = $1 AND "users_players"."deleted_at" IS NULL`,
			)).WithArgs(playerId).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			ret, err := r.ExistsActiveByPlayerId(context.Background(), playerId)

			require.NoError(t, err)
			require.True(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_有効な紐付けが無ければfalseを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserPlayer(db)

			mock.ExpectQuery(`SELECT count\(\*\) FROM "users_players"`).
				WithArgs(playerId).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			ret, err := r.ExistsActiveByPlayerId(context.Background(), playerId)

			require.NoError(t, err)
			require.False(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("正常系_紐付けを保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserPlayer(db)

			createdAt := time.Now().Local()

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "users_players" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"user_id"=$4,"player_id"=$5 WHERE "users_players"."deleted_at" IS NULL AND "id" = $6`,
			)).WithArgs(
				createdAt, AnyTime{}, gorm.DeletedAt{}, uid, playerId, id,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			userPlayer := entity.NewUserPlayer(id, createdAt, uid, playerId)

			require.NoError(t, r.Save(context.Background(), userPlayer))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_紐付けを論理削除する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserPlayer(db)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "users_players" SET "deleted_at"=$1 WHERE id = $2 AND "users_players"."deleted_at" IS NULL`,
			)).WithArgs(
				AnyTime{}, id,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.Delete(context.Background(), id))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
