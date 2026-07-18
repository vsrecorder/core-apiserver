package infrastructure

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

var notificationColumns = []string{
	"id", "created_at", "user_id", "category", "title", "body", "link_url", "is_read", "read_at",
}

func TestNotificationInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	createdAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.Local)

	t.Run("Save", func(t *testing.T) {
		t.Run("正常系_未読の通知はread_atをNULLで保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "notifications"`).WithArgs(
				id, createdAt, uid, "badge", "タイトル", "本文", "/badges", false, nil,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			n := entity.NewNotification(id, createdAt, uid, "badge", "タイトル", "本文", "/badges")

			require.NoError(t, r.Save(context.Background(), n))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_既読の通知はread_atも保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			readAt := createdAt.Add(time.Hour)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "notifications"`).WithArgs(
				id, createdAt, uid, "badge", "タイトル", "本文", "/badges", true, readAt,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			n := entity.NewNotification(id, createdAt, uid, "badge", "タイトル", "本文", "/badges")
			n.IsRead = true
			n.ReadAt = readAt

			require.NoError(t, r.Save(context.Background(), n))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_保存エラーをそのまま返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "notifications"`).WillReturnError(sql.ErrConnDone)
			mock.ExpectRollback()

			n := entity.NewNotification(id, createdAt, uid, "badge", "タイトル", "本文", "/badges")

			require.Error(t, r.Save(context.Background(), n))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("UpdateContent", func(t *testing.T) {
		t.Run("正常系_タイトルと本文と既読状態を更新する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			// map指定のUpdatesはキーのアルファベット順にSET句が組み立てられる
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "notifications" SET "body"=$1,"created_at"=$2,"is_read"=$3,"title"=$4 WHERE id = $5`,
			)).WithArgs(
				"新しい本文", createdAt, true, "新しいタイトル", id,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			err := r.UpdateContent(context.Background(), id, createdAt, "新しいタイトル", "新しい本文", true)

			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_更新対象がなければErrRecordNotFoundを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "notifications" SET`).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()

			err := r.UpdateContent(context.Background(), id, createdAt, "新しいタイトル", "新しい本文", true)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_作成日時とIDの降順で指定件数の通知を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			readAt := createdAt.Add(time.Hour)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "notifications" WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`,
			)).WithArgs(uid, 10).WillReturnRows(
				sqlmock.NewRows(notificationColumns).AddRow(
					"01HD7Y3K8D6FDHMHTZ2GT41TN3", createdAt, uid, "designation", "称号獲得", "本文2", "", true, &readAt,
				).AddRow(
					id, createdAt, uid, "badge", "タイトル", "本文", "/badges", false, nil,
				),
			)

			ret, err := r.FindByUserId(context.Background(), uid, 10)

			require.NoError(t, err)
			require.Len(t, ret, 2)
			require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN3", ret[0].ID)
			require.True(t, ret[0].IsRead)
			require.Equal(t, readAt, ret[0].ReadAt)
			require.Equal(t, id, ret[1].ID)
			require.False(t, ret[1].IsRead)
			require.True(t, ret[1].ReadAt.IsZero())
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectQuery(`SELECT \* FROM "notifications"`).
				WithArgs(uid, 10).WillReturnRows(sqlmock.NewRows(notificationColumns))

			ret, err := r.FindByUserId(context.Background(), uid, 10)

			require.NoError(t, err)
			require.Empty(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("CountUnreadByUserId", func(t *testing.T) {
		t.Run("正常系_未読の通知数を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT count(*) FROM "notifications" WHERE user_id = $1 AND is_read = $2`,
			)).WithArgs(uid, false).WillReturnRows(
				sqlmock.NewRows([]string{"count"}).AddRow(3),
			)

			count, err := r.CountUnreadByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Equal(t, 3, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("MarkAsRead", func(t *testing.T) {
		t.Run("正常系_本人の通知を既読にして既読日時を設定する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "notifications" SET "is_read"=$1,"read_at"=$2 WHERE id = $3 AND user_id = $4`,
			)).WithArgs(
				true, AnyTime{}, id, uid,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.MarkAsRead(context.Background(), id, uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_他人の通知や存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "notifications" SET`).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()

			err := r.MarkAsRead(context.Background(), id, uid)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("MarkAllAsReadByUserId", func(t *testing.T) {
		t.Run("正常系_未読の全通知を既読にする", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "notifications" SET "is_read"=$1,"read_at"=$2 WHERE user_id = $3 AND is_read = $4`,
			)).WithArgs(
				true, AnyTime{}, uid, false,
			).WillReturnResult(sqlmock.NewResult(0, 5))
			mock.ExpectCommit()

			require.NoError(t, r.MarkAllAsReadByUserId(context.Background(), uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_未読が0件でもエラーにしない", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewNotification(db)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "notifications" SET`).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()

			require.NoError(t, r.MarkAllAsReadByUserId(context.Background(), uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
