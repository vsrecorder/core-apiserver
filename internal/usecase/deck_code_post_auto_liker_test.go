package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestDeckCodePostAutoLiker(t *testing.T) {
	const officialUserId = "lgH4owuYpwNtQJVhTNrKMNjG2jM2"

	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.Local)

	posts := []*entity.DeckCodePost{
		{ID: "post-1", UserId: "owner-1"},
		{ID: "post-2", UserId: "owner-2"},
	}

	setup := func(t *testing.T, likerUserId string) (*mock_repository.MockDeckCodePostInterface, DeckCodePostAutoLikerInterface) {
		t.Helper()
		ctrl := gomock.NewController(t)
		postRepo := mock_repository.NewMockDeckCodePostInterface(ctrl)

		return postRepo, NewDeckCodePostAutoLiker(postRepo, likerUserId)
	}

	t.Run("正常系_未いいねの公開中の投稿に公式アカウントでいいねを付ける", func(t *testing.T) {
		overrideTimeNow(t, now)
		postRepo, autoLiker := setup(t, officialUserId)

		postRepo.EXPECT().FindActiveNotLikedBy(gomock.Any(), officialUserId, "", time.Time{}, 200).Return(posts, nil)
		postRepo.EXPECT().Like(gomock.Any(), "post-1", officialUserId, now).Return(nil)
		postRepo.EXPECT().Like(gomock.Any(), "post-2", officialUserId, now).Return(nil)

		count, err := autoLiker.LikeUnliked(context.Background(), "", time.Time{}, 200, false)

		require.NoError(t, err)
		require.Equal(t, 2, count)
	})

	t.Run("正常系_投稿者と公開日の絞り込みをリポジトリへ渡す", func(t *testing.T) {
		overrideTimeNow(t, now)
		postRepo, autoLiker := setup(t, officialUserId)

		publishedFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		postRepo.EXPECT().FindActiveNotLikedBy(gomock.Any(), officialUserId, "owner-1", publishedFrom, 50).Return(posts[:1], nil)
		postRepo.EXPECT().Like(gomock.Any(), "post-1", officialUserId, now).Return(nil)

		count, err := autoLiker.LikeUnliked(context.Background(), "owner-1", publishedFrom, 50, false)

		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("正常系_dry-runでは件数だけ返していいねを付けない", func(t *testing.T) {
		postRepo, autoLiker := setup(t, officialUserId)

		postRepo.EXPECT().FindActiveNotLikedBy(gomock.Any(), officialUserId, "", time.Time{}, 200).Return(posts, nil)

		count, err := autoLiker.LikeUnliked(context.Background(), "", time.Time{}, 200, true)

		require.NoError(t, err)
		require.Equal(t, 2, count)
	})

	t.Run("正常系_対象が無ければ何もしない", func(t *testing.T) {
		postRepo, autoLiker := setup(t, officialUserId)

		postRepo.EXPECT().FindActiveNotLikedBy(gomock.Any(), officialUserId, "", time.Time{}, 200).Return([]*entity.DeckCodePost{}, nil)

		count, err := autoLiker.LikeUnliked(context.Background(), "", time.Time{}, 200, false)

		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("異常系_公式アカウントが未設定なら対象を引かずにエラーを返す", func(t *testing.T) {
		_, autoLiker := setup(t, "")

		count, err := autoLiker.LikeUnliked(context.Background(), "", time.Time{}, 200, false)

		require.ErrorIs(t, err, errAutoLikerUserIdEmpty)
		require.Equal(t, 0, count)
	})

	t.Run("異常系_途中で失敗したらそこまでの件数とエラーを返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		postRepo, autoLiker := setup(t, officialUserId)

		wantErr := errors.New("db error")
		postRepo.EXPECT().FindActiveNotLikedBy(gomock.Any(), officialUserId, "", time.Time{}, 200).Return(posts, nil)
		postRepo.EXPECT().Like(gomock.Any(), "post-1", officialUserId, now).Return(nil)
		postRepo.EXPECT().Like(gomock.Any(), "post-2", officialUserId, now).Return(wantErr)

		count, err := autoLiker.LikeUnliked(context.Background(), "", time.Time{}, 200, false)

		require.ErrorIs(t, err, wantErr)
		require.Equal(t, 1, count, "残りは次の実行が拾うため、付けられた分だけ返す")
	})
}
