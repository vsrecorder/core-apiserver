package infrastructure

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// userGymViewColumns は user_gyms に shops / prefectures を JOIN した結果のカラム。
var userGymViewColumns = append(append([]string{}, shopColumns...), "created_at")

func userGymQuery(tail string) string {
	return `(?s)SELECT.*FROM "user_gyms".*JOIN shops ON shops\.id = user_gyms\.shop_id.*` +
		`LEFT JOIN prefectures ON prefectures\.id = shops\.prefecture_id.*` + regexp.QuoteMeta(tail)
}

func TestUserGymInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	createdAt := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_登録順に店舗情報つきで返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			rows := sqlmock.NewRows(userGymViewColumns).AddRow(
				uint(10317), "古本市場新琴似店", "001-0907", uint(1), "北海道",
				"札幌市北区新琴似七条１７−１", "011-000-0000", "10:00~20:00", "https://example.com",
				createdAt,
			)

			mock.ExpectQuery(userGymQuery(
				`WHERE user_gyms.user_id = $1 ORDER BY user_gyms.created_at ASC, user_gyms.shop_id ASC`,
			)).WithArgs(uid).WillReturnRows(rows)

			views, err := r.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Len(t, views, 1)
			require.Equal(t, uint(10317), views[0].Shop.ID)
			require.Equal(t, "北海道", views[0].Shop.PrefectureName)
			require.Equal(t, createdAt, views[0].CreatedAt)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		// 店舗マスタから消えた店への登録は JOIN で落ちる。行が無くても空スライスを返す。
		t.Run("正常系_1件も無ければ空スライスを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			mock.ExpectQuery(userGymQuery(`WHERE user_gyms.user_id = $1`)).
				WithArgs(uid).
				WillReturnRows(sqlmock.NewRows(userGymViewColumns))

			views, err := r.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Empty(t, views)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("LockByUserId", func(t *testing.T) {
		// 上限チェックを直列化するためのユーザ単位ロック。行が無くても取れる必要がある。
		t.Run("正常系_ユーザ単位のアドバイザリロックを取る", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			mock.ExpectExec(regexp.QuoteMeta(`SELECT pg_advisory_xact_lock(hashtext($1))`)).
				WithArgs(uid).
				WillReturnResult(sqlmock.NewResult(0, 0))

			require.NoError(t, r.LockByUserId(context.Background(), uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_user_idとshop_idと登録日時を保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_gyms"`).
				WithArgs(uid, uint(10317), createdAt).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			err := r.Create(context.Background(), entity.NewUserGym(uid, 10317, createdAt))

			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		// 同じ店舗への同時登録は主キー違反になる。500ではなく「登録済み」として扱う。
		t.Run("異常系_主キー違反はErrAlreadyExistsへ変換して返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_gyms"`).
				WillReturnError(&pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"})
			mock.ExpectRollback()

			err := r.Create(context.Background(), entity.NewUserGym(uid, 10317, createdAt))

			require.ErrorIs(t, err, apperror.ErrAlreadyExists)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_保存エラーをそのまま返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_gyms"`).WillReturnError(sql.ErrConnDone)
			mock.ExpectRollback()

			err := r.Create(context.Background(), entity.NewUserGym(uid, 10317, createdAt))

			require.ErrorIs(t, err, sql.ErrConnDone)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_user_idとshop_idの行を物理削除する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "user_gyms" WHERE user_id = $1 AND shop_id = $2`)).
				WithArgs(uid, uint(10317)).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.Delete(context.Background(), uid, 10317))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("DeleteByUserId", func(t *testing.T) {
		t.Run("正常系_そのユーザの登録をまとめて物理削除する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserGym(db)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "user_gyms" WHERE user_id = $1`)).
				WithArgs(uid).
				WillReturnResult(sqlmock.NewResult(0, 3))
			mock.ExpectCommit()

			require.NoError(t, r.DeleteByUserId(context.Background(), uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
