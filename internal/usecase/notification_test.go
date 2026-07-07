package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func newNotificationTestUsecase(mockCtrl *gomock.Controller) (
	NotificationInterface,
	*mock_repository.MockNotificationInterface,
) {
	repo := mock_repository.NewMockNotificationInterface(mockCtrl)
	return NewNotification(repo), repo
}

func TestNotification_ListByUserId(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	u, repo := newNotificationTestUsecase(mockCtrl)

	expected := []*entity.Notification{
		entity.NewNotification("n-1", time.Now(), "user-1", NotificationCategoryBadge, "title", "body", "/users"),
	}
	repo.EXPECT().FindByUserId(gomock.Any(), "user-1", 20).Return(expected, nil)

	got, err := u.ListByUserId(context.Background(), "user-1", 20)

	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestNotification_CountUnreadByUserId(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	u, repo := newNotificationTestUsecase(mockCtrl)

	repo.EXPECT().CountUnreadByUserId(gomock.Any(), "user-1").Return(3, nil)

	got, err := u.CountUnreadByUserId(context.Background(), "user-1")

	require.NoError(t, err)
	require.Equal(t, 3, got)
}

func TestNotification_MarkAsRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	u, repo := newNotificationTestUsecase(mockCtrl)

	// userIdはrepository層で本人の通知かどうかの絞り込みに使われるため、idと一緒に渡す
	repo.EXPECT().MarkAsRead(gomock.Any(), "n-1", "user-1").Return(nil)

	err := u.MarkAsRead(context.Background(), "user-1", "n-1")

	require.NoError(t, err)
}

func TestNotification_MarkAllAsRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	u, repo := newNotificationTestUsecase(mockCtrl)

	repo.EXPECT().MarkAllAsReadByUserId(gomock.Any(), "user-1").Return(nil)

	err := u.MarkAllAsRead(context.Background(), "user-1")

	require.NoError(t, err)
}
