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

var unofficialEventColumns = []string{
	"id", "created_at", "updated_at", "deleted_at", "user_id", "title", "date",
}

func TestUnofficialEventInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDの自由形式イベントを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUnofficialEvent(db)

			now := time.Now().Local()

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "unofficial_events" WHERE id = $1 AND "unofficial_events"."deleted_at" IS NULL ORDER BY "unofficial_events"."id" LIMIT $2`,
			)).WithArgs(id, 1).WillReturnRows(
				sqlmock.NewRows(unofficialEventColumns).AddRow(
					id, now, now, gorm.DeletedAt{}, uid, "自主大会", date,
				),
			)

			ret, err := r.FindById(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, "自主大会", ret.Title)
			require.Equal(t, date, ret.Date)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundへ変換する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUnofficialEvent(db)

			mock.ExpectQuery(`SELECT \* FROM "unofficial_events"`).
				WithArgs(id, 1).WillReturnRows(sqlmock.NewRows(unofficialEventColumns))

			ret, err := r.FindById(context.Background(), id)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("正常系_イベントを保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUnofficialEvent(db)

			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "unofficial_events" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"user_id"=$4,"title"=$5,"date"=$6 WHERE "unofficial_events"."deleted_at" IS NULL AND "id" = $7`,
			)).WithArgs(
				AnyTime{}, AnyTime{}, gorm.DeletedAt{}, uid, "自主大会", date, id,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			event := entity.NewUnofficialEvent(id, uid, "自主大会", date)

			require.NoError(t, r.Save(context.Background(), event))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_作成日時を保持したイベントはその値のまま保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUnofficialEvent(db)

			createdAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local)

			// GORMのSaveは全カラムを書き戻すため、created_atを渡さないと更新時刻で潰れる
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "unofficial_events" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"user_id"=$4,"title"=$5,"date"=$6 WHERE "unofficial_events"."deleted_at" IS NULL AND "id" = $7`,
			)).WithArgs(
				createdAt, AnyTime{}, gorm.DeletedAt{}, uid, "身内対戦会", date, id,
			).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			event := entity.NewUnofficialEvent(id, uid, "身内対戦会", date)
			event.CreatedAt = createdAt

			require.NoError(t, r.Save(context.Background(), event))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_イベントを削除する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUnofficialEvent(db)

			// unofficial_events は論理削除(deleted_at)
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(
				`UPDATE "unofficial_events" SET "deleted_at"=$1 WHERE id = $2 AND "unofficial_events"."deleted_at" IS NULL`,
			)).WithArgs(AnyTime{}, id).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.Delete(context.Background(), id))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
