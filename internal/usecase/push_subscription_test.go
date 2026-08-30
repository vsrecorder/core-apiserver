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

func TestPushSubscriptionUsecase(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.Local)

	t.Run("Subscribe_正常系_IDと時刻を採番しplatformを正規化してUpsertする", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)
		var saved *entity.PushSubscription
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, s *entity.PushSubscription) error {
				saved = s
				return nil
			},
		)

		err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/1", "p256dh", "auth", "unknown-platform")

		require.NoError(t, err)
		require.NotEmpty(t, saved.ID)
		require.Equal(t, now, saved.CreatedAt)
		require.Equal(t, "user-1", saved.UserId)
		require.Equal(t, "https://push.example.com/1", saved.Endpoint)
		require.Equal(t, "", saved.Platform) // 未知の platform は空文字に丸める
	})

	t.Run("Subscribe_異常系_保存エラーをそのまま返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

		require.Error(t, u.Subscribe(context.Background(), "user-1", "https://push.example.com/1", "p", "a", entity.PushPlatformDesktop))
	})

	t.Run("Subscribe_異常系_生きている購読が上限に達していれば新しいendpointは受け付けない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		live := make([]*entity.PushSubscription, 0, pushSubscriptionsPerUserLimit)
		for i := 0; i < pushSubscriptionsPerUserLimit; i++ {
			live = append(live, entity.NewPushSubscription("s", now, "user-1", "https://push.example.com/"+string(rune('a'+i)), "p", "a", ""))
		}
		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(live, nil)
		// Upsert は呼ばれない

		err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/new", "p", "a", entity.PushPlatformDesktop)

		require.ErrorIs(t, err, apperror.ErrTooManyPushSubscriptions)
	})

	t.Run("Subscribe_正常系_上限に達していても既存endpointの再購読(更新)は通す", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		live := make([]*entity.PushSubscription, 0, pushSubscriptionsPerUserLimit)
		for i := 0; i < pushSubscriptionsPerUserLimit; i++ {
			live = append(live, entity.NewPushSubscription("s", now, "user-1", "https://push.example.com/"+string(rune('a'+i)), "p", "a", ""))
		}
		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(live, nil)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

		require.NoError(t, u.Subscribe(context.Background(), "user-1", "https://push.example.com/a", "p", "a", entity.PushPlatformDesktop))
	})

	t.Run("Unsubscribe_正常系_本人のendpointを現在時刻で解除する", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		repo.EXPECT().RevokeByUserIdAndEndpoint(gomock.Any(), "user-1", "https://push.example.com/1", now).Return(nil)

		require.NoError(t, u.Unsubscribe(context.Background(), "user-1", "https://push.example.com/1"))
	})
}
