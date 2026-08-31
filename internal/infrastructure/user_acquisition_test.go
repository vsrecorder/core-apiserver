package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func TestUserAcquisitionInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	createdAt := time.Date(2026, 8, 31, 21, 0, 0, 0, time.Local)
	landingAt := time.Date(2026, 8, 30, 12, 0, 0, 0, time.Local)

	newEntity := func() *entity.UserAcquisition {
		a := entity.NewUserAcquisition(uid, createdAt)
		a.Source = "x"
		a.Medium = "social"
		a.Campaign = entity.AcquisitionCampaignHowtoCta
		a.Content = "20260831a"
		a.Referrer = "t.co"
		a.LandingPath = "/records/quick"
		a.LandingAt = landingAt

		return a
	}

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_流入元を保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserAcquisition(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_acquisitions" .* ON CONFLICT \("user_id"\) DO NOTHING`).
				WithArgs(
					uid, "x", "social", entity.AcquisitionCampaignHowtoCta, "20260831a",
					"t.co", "/records/quick", landingAt, false, nil, createdAt, createdAt,
				).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.Create(context.Background(), newEntity()))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_判明しなかった項目はNULLで保存する", func(t *testing.T) {
			// 空文字で埋めると Grafana 側の COALESCE(campaign, ...) が効かず、
			// 「タグ無しの直接流入」と区別が付かなくなる
			db, mock := setupSqlmockDB(t)
			r := NewUserAcquisition(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_acquisitions"`).
				WithArgs(
					uid, "x", nil, nil, nil,
					nil, nil, nil, true, nil, createdAt, createdAt,
				).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			a := entity.NewUserAcquisition(uid, createdAt)
			a.Source = "x"
			a.SourceInferred = true

			require.NoError(t, r.Create(context.Background(), a))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_二重に届いても既存の行を上書きしない", func(t *testing.T) {
			// 初回タッチを保つため、衝突は無視して先に入っている行を残す
			db, mock := setupSqlmockDB(t)
			r := NewUserAcquisition(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_acquisitions" .* ON CONFLICT \("user_id"\) DO NOTHING`).
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()

			require.NoError(t, r.Create(context.Background(), newEntity()))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_保存エラーをそのまま返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserAcquisition(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "user_acquisitions"`).WillReturnError(sql.ErrConnDone)
			mock.ExpectRollback()

			require.ErrorIs(t, r.Create(context.Background(), newEntity()), sql.ErrConnDone)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("DeleteByUserId", func(t *testing.T) {
		t.Run("正常系_退会時に行ごと削除する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewUserAcquisition(db)

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "user_acquisitions" WHERE user_id = \$1`).
				WithArgs(uid).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.DeleteByUserId(context.Background(), uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
