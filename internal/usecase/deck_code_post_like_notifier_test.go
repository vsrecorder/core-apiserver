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

func TestDeckCodePostLikeNotifier(t *testing.T) {
	now := time.Date(2026, 9, 4, 8, 0, 0, 0, time.Local)
	day := now.AddDate(0, 0, -1)
	from := time.Date(2026, 9, 3, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 9, 4, 0, 0, 0, 0, time.Local)

	digests := []*entity.DeckCodePostLikeDigest{
		{PostId: "post-1", OwnerUserId: "owner-1", DeckName: "ドラパルト ヨノワール型", LikeCount: 3, LatestLikerName: "タイチ"},
		{PostId: "post-2", OwnerUserId: "owner-2", DeckName: "メガサーナイト", LikeCount: 1, LatestLikerName: "さくら"},
	}

	setup := func(t *testing.T) (*mock_repository.MockDeckCodePostInterface, *mock_repository.MockNotificationInterface, *stubPushNotifier, DeckCodePostLikeNotifierInterface) {
		t.Helper()
		ctrl := gomock.NewController(t)
		postRepo := mock_repository.NewMockDeckCodePostInterface(ctrl)
		notificationRepo := mock_repository.NewMockNotificationInterface(ctrl)
		push := &stubPushNotifier{}
		return postRepo, notificationRepo, push, NewDeckCodePostLikeNotifier(postRepo, notificationRepo, push)
	}

	t.Run("正常系_投稿ごとに1通の定型文を作り push も送る", func(t *testing.T) {
		overrideTimeNow(t, now)
		postRepo, notificationRepo, push, notifier := setup(t)

		postRepo.EXPECT().FindLikeDigests(gomock.Any(), from, to).Return(digests, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "owner-1", NotificationCategoryLike, "/shared_decks/post-1?d=2026-09-03").Return(false, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "owner-2", NotificationCategoryLike, "/shared_decks/post-2?d=2026-09-03").Return(false, nil)

		var saved []*entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				saved = append(saved, n)
				return nil
			},
		).Times(2)

		count, err := notifier.NotifyDay(context.Background(), day, "", false)

		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Len(t, saved, 2)
		require.Equal(t, "owner-1", saved[0].UserId)
		require.Equal(t, NotificationCategoryLike, saved[0].Category)
		require.Equal(t, "いいねが届きました", saved[0].Title)
		require.Equal(t, "タイチさんほか2人が「ドラパルト ヨノワール型」にいいねしました", saved[0].Body)
		require.Equal(t, "/shared_decks/post-1?d=2026-09-03", saved[0].LinkUrl)
		require.Equal(t, now, saved[0].CreatedAt)
		require.Equal(t, "さくらさんが「メガサーナイト」にいいねしました", saved[1].Body, "1人のときは「ほか」を付けない")
		require.Len(t, push.calls, 2)
		require.Equal(t, PushCampaignDeckCodePostLike, push.campaigns()[0])
	})

	t.Run("正常系_同じ投稿・同じ日の通知は二重に作らない", func(t *testing.T) {
		overrideTimeNow(t, now)
		postRepo, notificationRepo, push, notifier := setup(t)

		postRepo.EXPECT().FindLikeDigests(gomock.Any(), from, to).Return(digests[:1], nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "owner-1", NotificationCategoryLike, "/shared_decks/post-1?d=2026-09-03").Return(true, nil)

		count, err := notifier.NotifyDay(context.Background(), day, "", false)

		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Empty(t, push.calls)
	})

	t.Run("正常系_dry-runでは件数だけ返して通知を作らない", func(t *testing.T) {
		postRepo, notificationRepo, push, notifier := setup(t)

		postRepo.EXPECT().FindLikeDigests(gomock.Any(), from, to).Return(digests, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), gomock.Any(), NotificationCategoryLike, gomock.Any()).Return(false, nil).Times(2)

		count, err := notifier.NotifyDay(context.Background(), day, "", true)

		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Empty(t, push.calls)
	})

	t.Run("正常系_いいねが無い日は何もしない", func(t *testing.T) {
		postRepo, _, push, notifier := setup(t)

		postRepo.EXPECT().FindLikeDigests(gomock.Any(), from, to).Return([]*entity.DeckCodePostLikeDigest{}, nil)

		count, err := notifier.NotifyDay(context.Background(), day, "", false)

		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Empty(t, push.calls)
	})
}
