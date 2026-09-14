package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestPushDeliveryUsecase(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.Local)

	// 配達ログ1件。既読にする対象の通知(notification_id)を持つ
	delivery := func(notificationId string) *entity.PushDelivery {
		return entity.NewPushDelivery(
			"d-1",
			now,
			"user-1",
			"sub-1",
			notificationId,
			"weekly_report",
			"sent",
			201,
		)
	}

	t.Run("MarkDelivered_正常系_本人のidを現在時刻で記録する", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewPushDelivery(repo, notificationRepo)

		repo.EXPECT().MarkDelivered(gomock.Any(), "d-1", "user-1", now).Return(nil)

		require.NoError(t, u.MarkDelivered(context.Background(), "user-1", "d-1"))
	})

	t.Run("MarkClicked_正常系_記録して元になった通知を既読にする", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewPushDelivery(repo, notificationRepo)

		repo.EXPECT().MarkClicked(gomock.Any(), "d-1", "user-1", now).Return(nil)
		repo.EXPECT().FindById(gomock.Any(), "d-1", "user-1").Return(delivery("n-1"), nil)
		// push を開いた時点で本人はその知らせを見ているので、ベルの未読を残さない
		notificationRepo.EXPECT().MarkAsRead(gomock.Any(), "n-1", "user-1").Return(nil)

		require.NoError(t, u.MarkClicked(context.Background(), "user-1", "d-1"))
	})

	t.Run("MarkClicked_正常系_通知を伴わない配達では既読化しない", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewPushDelivery(repo, notificationRepo)

		repo.EXPECT().MarkClicked(gomock.Any(), "d-1", "user-1", now).Return(nil)
		repo.EXPECT().FindById(gomock.Any(), "d-1", "user-1").Return(delivery(""), nil)

		require.NoError(t, u.MarkClicked(context.Background(), "user-1", "d-1"))
	})

	t.Run("MarkClicked_正常系_通知が見つからなくてもタップの記録は成功にする", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewPushDelivery(repo, notificationRepo)

		repo.EXPECT().MarkClicked(gomock.Any(), "d-1", "user-1", now).Return(nil)
		repo.EXPECT().FindById(gomock.Any(), "d-1", "user-1").Return(delivery("n-1"), nil)
		notificationRepo.EXPECT().MarkAsRead(gomock.Any(), "n-1", "user-1").Return(apperror.ErrRecordNotFound)

		require.NoError(t, u.MarkClicked(context.Background(), "user-1", "d-1"))
	})

	t.Run("MarkClicked_正常系_既読化が失敗してもタップの記録は成功にする", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewPushDelivery(repo, notificationRepo)

		repo.EXPECT().MarkClicked(gomock.Any(), "d-1", "user-1", now).Return(nil)
		repo.EXPECT().FindById(gomock.Any(), "d-1", "user-1").Return(nil, errors.New("db error"))

		require.NoError(t, u.MarkClicked(context.Background(), "user-1", "d-1"))
	})

	t.Run("MarkClicked_異常系_他人のidはErrRecordNotFoundをそのまま返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
		u := NewPushDelivery(repo, notificationRepo)

		// 記録できていないので既読化もしない(他人の通知に触れない)
		repo.EXPECT().MarkClicked(gomock.Any(), "d-1", "user-2", now).Return(apperror.ErrRecordNotFound)

		require.ErrorIs(t, u.MarkClicked(context.Background(), "user-2", "d-1"), apperror.ErrRecordNotFound)
	})
}
