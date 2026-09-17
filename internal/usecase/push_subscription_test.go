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
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(nil, apperror.ErrRecordNotFound)
		var saved *entity.PushSubscription
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, s *entity.PushSubscription) error {
				saved = s
				return nil
			},
		)

		wasRevoked, err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/1", "p256dh", "auth", "unknown-platform")

		require.NoError(t, err)
		require.False(t, wasRevoked) // 初めての購読は「復活」ではない
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
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(nil, apperror.ErrRecordNotFound)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

		_, err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/1", "p", "a", entity.PushPlatformDesktop)

		require.Error(t, err)
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

		_, err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/new", "p", "a", entity.PushPlatformDesktop)

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
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/a").Return(live[0], nil)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

		_, err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/a", "p", "a", entity.PushPlatformDesktop)

		require.NoError(t, err)
	})

	// 同じ端末でのアカウント切替(A がログアウトし B がログイン)。ブラウザの購読は同じなので
	// endpoint と鍵は同じまま持ち主だけが変わる。これは通す。
	t.Run("Subscribe_正常系_同じ端末での持ち主変更は鍵が一致すれば通す", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		previous := entity.NewPushSubscription("s", now, "user-a", "https://push.example.com/1", "p256dh", "auth", "")
		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-b").Return(nil, nil)
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(previous, nil)
		var saved *entity.PushSubscription
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, s *entity.PushSubscription) error {
				saved = s
				return nil
			},
		)

		_, err := u.Subscribe(context.Background(), "user-b", "https://push.example.com/1", "p256dh", "auth", "")

		require.NoError(t, err)
		require.Equal(t, "user-b", saved.UserId)
	})

	// endpoint だけを知る第三者が自分のアカウントで登録し直すと、元の持ち主のその端末への
	// 通知が止まる。鍵が違う(=同じブラウザの購読ではない)持ち主変更は弾き、Upsert しない。
	t.Run("Subscribe_異常系_他人のendpointを別の鍵で登録し直すとErrPushSubscriptionOwnedByOther", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		previous := entity.NewPushSubscription("s", now, "user-a", "https://push.example.com/1", "p256dh", "auth", "")
		repo.EXPECT().FindLiveByUserId(gomock.Any(), "attacker").Return(nil, nil)
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(previous, nil)
		// Upsert は EXPECT しない(弾かれて呼ばれない)

		wasRevoked, err := u.Subscribe(context.Background(), "attacker", "https://push.example.com/1", "other-p256dh", "other-auth", "")

		require.ErrorIs(t, err, apperror.ErrPushSubscriptionOwnedByOther)
		require.False(t, wasRevoked)
	})

	// auth だけ一致しても鍵は一致とみなさない(両方が同じブラウザの購読であることの証明)
	t.Run("Subscribe_異常系_鍵の片方だけ一致する持ち主変更も弾く", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		previous := entity.NewPushSubscription("s", now, "user-a", "https://push.example.com/1", "p256dh", "auth", "")
		repo.EXPECT().FindLiveByUserId(gomock.Any(), "attacker").Return(nil, nil)
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(previous, nil)

		_, err := u.Subscribe(context.Background(), "attacker", "https://push.example.com/1", "other-p256dh", "auth", "")

		require.ErrorIs(t, err, apperror.ErrPushSubscriptionOwnedByOther)
	})

	// 本人が自分の endpoint を登録し直すのは、鍵が変わっていても常に許す
	t.Run("Subscribe_正常系_本人の再登録は鍵が違っても通す", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		previous := entity.NewPushSubscription("s", now, "user-1", "https://push.example.com/1", "old-p256dh", "old-auth", "")
		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(previous, nil)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

		_, err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/1", "new-p256dh", "new-auth", "")

		require.NoError(t, err)
	})

	// Upsert は revoked_at を消して購読を復活させてしまうため、失効していた事実は
	// ここで返さないと端末へ伝わらない。端末には購読オブジェクトが残っていて、
	// 端末側だけでは「サーバから失効させられた」ことに気付けない
	t.Run("Subscribe_正常系_失効していたendpointの再購読ではwasRevokedがtrueになる", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		revoked := entity.NewPushSubscription("s", now, "user-1", "https://push.example.com/1", "p", "a", "")
		revoked.RevokedAt = now

		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(revoked, nil)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

		wasRevoked, err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/1", "p", "a", entity.PushPlatformDesktop)

		require.NoError(t, err)
		require.True(t, wasRevoked)
	})

	t.Run("Subscribe_正常系_生きているendpointの再購読ではwasRevokedはfalse", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushSubscriptionInterface(mockCtrl)
		u := NewPushSubscription(repo)

		alive := entity.NewPushSubscription("s", now, "user-1", "https://push.example.com/1", "p", "a", "")

		repo.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)
		repo.EXPECT().FindByEndpoint(gomock.Any(), "https://push.example.com/1").Return(alive, nil)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

		wasRevoked, err := u.Subscribe(context.Background(), "user-1", "https://push.example.com/1", "p", "a", entity.PushPlatformDesktop)

		require.NoError(t, err)
		require.False(t, wasRevoked)
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
