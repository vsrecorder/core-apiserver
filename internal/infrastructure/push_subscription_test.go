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

var pushSubscriptionColumns = []string{
	"id", "created_at", "updated_at", "revoked_at", "user_id", "endpoint", "p256dh", "auth", "platform", "failure_count", "last_success_at",
}

func TestPushSubscriptionInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	endpoint := "https://fcm.googleapis.com/fcm/send/abc"
	createdAt := time.Date(2026, 8, 29, 12, 0, 0, 0, time.Local)

	t.Run("Upsert", func(t *testing.T) {
		t.Run("正常系_endpointで衝突したら鍵と持ち主を更新しrevoked_atを戻す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "push_subscriptions" .* ON CONFLICT \("endpoint"\) DO UPDATE SET`).
				WithArgs(
					// INSERT の値
					id, createdAt, AnyTime{}, nil, uid, endpoint, "p256dh-key", "auth-key", entity.PushPlatformAndroid, 0, nil,
					// DO UPDATE SET の値(列名の昇順)
					"auth-key", 0, "p256dh-key", entity.PushPlatformAndroid, nil, AnyTime{}, uid,
				).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			s := entity.NewPushSubscription(id, createdAt, uid, endpoint, "p256dh-key", "auth-key", entity.PushPlatformAndroid)

			require.NoError(t, r.Upsert(context.Background(), s))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_保存エラーをそのまま返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "push_subscriptions"`).WillReturnError(sql.ErrConnDone)
			mock.ExpectRollback()

			s := entity.NewPushSubscription(id, createdAt, uid, endpoint, "p256dh-key", "auth-key", "")

			require.ErrorIs(t, r.Upsert(context.Background(), s), sql.ErrConnDone)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindLiveByUserId", func(t *testing.T) {
		t.Run("正常系_解除されていない購読を作成順に返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			lastSuccess := createdAt.Add(time.Hour)
			rows := sqlmock.NewRows(pushSubscriptionColumns).
				AddRow(id, createdAt, createdAt, nil, uid, endpoint, "p256dh-key", "auth-key", entity.PushPlatformIOSPWA, 2, lastSuccess)

			mock.ExpectQuery(`SELECT \* FROM "push_subscriptions" WHERE user_id = \$1 AND revoked_at IS NULL ORDER BY created_at ASC`).
				WithArgs(uid).
				WillReturnRows(rows)

			got, err := r.FindLiveByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, id, got[0].ID)
			require.Equal(t, endpoint, got[0].Endpoint)
			require.Equal(t, entity.PushPlatformIOSPWA, got[0].Platform)
			require.Equal(t, 2, got[0].FailureCount)
			require.False(t, got[0].IsRevoked())
			require.Equal(t, lastSuccess, got[0].LastSuccessAt)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_購読が無ければ空スライスを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			mock.ExpectQuery(`SELECT \* FROM "push_subscriptions"`).
				WithArgs(uid).
				WillReturnRows(sqlmock.NewRows(pushSubscriptionColumns))

			got, err := r.FindLiveByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Empty(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("Revoke", func(t *testing.T) {
		t.Run("正常系_revoked_atとupdated_atを更新する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			at := createdAt.Add(2 * time.Hour)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_subscriptions" SET "revoked_at"=\$1,"updated_at"=\$2 WHERE id = \$3 AND revoked_at IS NULL`).
				WithArgs(at, at, id).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.Revoke(context.Background(), id, at))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_RevokeByUserIdAndEndpointは該当行が無くてもエラーにしない", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			at := createdAt.Add(2 * time.Hour)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_subscriptions" SET "revoked_at"=\$1,"updated_at"=\$2 WHERE user_id = \$3 AND endpoint = \$4 AND revoked_at IS NULL`).
				WithArgs(at, at, uid, endpoint).
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()

			require.NoError(t, r.RevokeByUserIdAndEndpoint(context.Background(), uid, endpoint, at))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("MarkSuccess_IncrementFailure", func(t *testing.T) {
		t.Run("正常系_成功でfailure_countを0に戻す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			at := createdAt.Add(time.Hour)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_subscriptions" SET "failure_count"=\$1,"last_success_at"=\$2,"updated_at"=\$3 WHERE id = \$4`).
				WithArgs(0, at, at, id).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.MarkSuccess(context.Background(), id, at))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_失敗でfailure_countを1増やす", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			at := createdAt.Add(time.Hour)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_subscriptions" SET "failure_count"=failure_count \+ 1,"updated_at"=\$1 WHERE id = \$2`).
				WithArgs(at, id).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.IncrementFailure(context.Background(), id, at))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("DeleteByUserId", func(t *testing.T) {
		t.Run("正常系_user_idで行ごと削除する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushSubscription(db)

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "push_subscriptions" WHERE user_id = \$1`).
				WithArgs(uid).
				WillReturnResult(sqlmock.NewResult(0, 2))
			mock.ExpectCommit()

			require.NoError(t, r.DeleteByUserId(context.Background(), uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
