package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

var pushDeliveryColumns = []string{
	"id", "created_at", "user_id", "subscription_id", "notification_id", "campaign", "status", "status_code", "delivered_at", "clicked_at",
}

func TestPushDeliveryInfrastructure(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	createdAt := time.Date(2026, 8, 28, 20, 0, 0, 0, time.Local)

	t.Run("Save", func(t *testing.T) {
		t.Run("正常系_未到達の配達ログはdelivered_at/clicked_atをNULLで保存する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "push_deliveries"`).
				WithArgs(id, createdAt, uid, "sub-1", "n-1", "weekend_reminder", entity.PushDeliveryStatusSent, 201, nil, nil).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			d := entity.NewPushDelivery(id, createdAt, uid, "sub-1", "n-1", "weekend_reminder", entity.PushDeliveryStatusSent, 201)

			require.NoError(t, r.Save(context.Background(), d))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("UpdateResult", func(t *testing.T) {
		t.Run("正常系_送出結果でstatusとstatus_codeを書く", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_deliveries" SET "status"=\$1,"status_code"=\$2 WHERE id = \$3`).
				WithArgs(entity.PushDeliveryStatusSent, 201, id).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.UpdateResult(context.Background(), id, entity.PushDeliveryStatusSent, 201))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_存在しないidはErrRecordNotFound", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_deliveries"`).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()

			require.ErrorIs(t, r.UpdateResult(context.Background(), "missing", entity.PushDeliveryStatusFailed, 503), apperror.ErrRecordNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("MarkDelivered_MarkClicked", func(t *testing.T) {
		at := createdAt.Add(time.Minute)

		t.Run("正常系_本人の配達ログならdelivered_atを入れる(2回目は最初の時刻を保つ)", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_deliveries" SET "delivered_at"=COALESCE\(delivered_at, \$1\) WHERE id = \$2 AND user_id = \$3`).
				WithArgs(at, id, uid).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			require.NoError(t, r.MarkDelivered(context.Background(), id, uid, at))
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_他人のidや存在しないidはErrRecordNotFound", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			mock.ExpectBegin()
			mock.ExpectExec(`UPDATE "push_deliveries" SET "clicked_at"=COALESCE\(clicked_at, \$1\) WHERE id = \$2 AND user_id = \$3`).
				WithArgs(at, id, "someone-else").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()

			require.ErrorIs(t, r.MarkClicked(context.Background(), id, "someone-else", at), apperror.ErrRecordNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("CountNotificationsByUserIdAndCampaignsSince", func(t *testing.T) {
		t.Run("正常系_受理された(sent)配達だけをnotification_idのdistinctで数える", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			since := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)

			mock.ExpectQuery(`SELECT COUNT\(DISTINCT\("notification_id"\)\) FROM "push_deliveries" WHERE user_id = \$1 AND campaign IN \(\$2,\$3\) AND status = \$4 AND created_at >= \$5`).
				WithArgs(uid, "weekend_reminder", "streak_nudge", entity.PushDeliveryStatusSent, since).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

			got, err := r.CountNotificationsByUserIdAndCampaignsSince(context.Background(), uid, []string{"weekend_reminder", "streak_nudge"}, since)

			require.NoError(t, err)
			require.Equal(t, 2, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_campaignsが空ならクエリを発行せず0を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			got, err := r.CountNotificationsByUserIdAndCampaignsSince(context.Background(), uid, nil, createdAt)

			require.NoError(t, err)
			require.Equal(t, 0, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindRecentByUserIdAndCampaign", func(t *testing.T) {
		t.Run("正常系_新しい順にlimit件返しclicked_atを詰め替える", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			clickedAt := createdAt.Add(time.Hour)
			rows := sqlmock.NewRows(pushDeliveryColumns).
				AddRow("d-2", createdAt, uid, "sub-1", "n-2", "weekend_reminder", entity.PushDeliveryStatusSent, 201, createdAt, clickedAt).
				AddRow("d-1", createdAt.AddDate(0, 0, -7), uid, "sub-1", "n-1", "weekend_reminder", entity.PushDeliveryStatusSent, 201, nil, nil)

			mock.ExpectQuery(`SELECT \* FROM "push_deliveries" WHERE user_id = \$1 AND campaign = \$2 ORDER BY created_at DESC, id DESC LIMIT \$3`).
				WithArgs(uid, "weekend_reminder", 20).
				WillReturnRows(rows)

			got, err := r.FindRecentByUserIdAndCampaign(context.Background(), uid, "weekend_reminder", 20)

			require.NoError(t, err)
			require.Len(t, got, 2)
			require.True(t, got[0].IsClicked())
			require.Equal(t, clickedAt, got[0].ClickedAt)
			require.False(t, got[1].IsClicked())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("DeleteByUserId", func(t *testing.T) {
		t.Run("正常系_user_idで行ごと削除する", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewPushDelivery(db)

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "push_deliveries" WHERE user_id = \$1`).
				WithArgs(uid).
				WillReturnResult(sqlmock.NewResult(0, 3))
			mock.ExpectCommit()

			require.NoError(t, r.DeleteByUserId(context.Background(), uid))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
