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

func TestActiveUserVerifier(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.Local)
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_登録済みのユーザーは有効", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockUserInterface(gomock.NewController(t))
		repo.EXPECT().FindById(gomock.Any(), uid).Return(&entity.User{ID: uid}, nil)

		active, err := NewActiveUserVerifier(repo).IsActiveUser(context.Background(), uid)

		require.NoError(t, err)
		require.True(t, active)
	})

	// FindById は論理削除済みを返さないため、退会済みと未登録はどちらも「見つからない」になる
	t.Run("正常系_未登録または退会済みのユーザーは無効", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockUserInterface(gomock.NewController(t))
		repo.EXPECT().FindById(gomock.Any(), uid).Return(nil, apperror.ErrRecordNotFound)

		active, err := NewActiveUserVerifier(repo).IsActiveUser(context.Background(), uid)

		require.NoError(t, err)
		require.False(t, active)
	})

	t.Run("異常系_リポジトリのエラーはそのまま返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockUserInterface(gomock.NewController(t))
		repo.EXPECT().FindById(gomock.Any(), uid).Return(nil, errors.New("db down"))

		active, err := NewActiveUserVerifier(repo).IsActiveUser(context.Background(), uid)

		require.Error(t, err)
		require.False(t, active)
	})

	// 認証のたびに users を引かないよう、有効と確認した結果は TTL のあいだ保持する
	t.Run("正常系_有効と確認した結果はTTLのあいだ保持しDBを引かない", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockUserInterface(gomock.NewController(t))
		repo.EXPECT().FindById(gomock.Any(), uid).Return(&entity.User{ID: uid}, nil).Times(1)
		verifier := NewActiveUserVerifier(repo)

		for i := 0; i < 3; i++ {
			active, err := verifier.IsActiveUser(context.Background(), uid)
			require.NoError(t, err)
			require.True(t, active)
		}
	})

	t.Run("正常系_TTLが過ぎたら再びDBで確認し退会が反映される", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockUserInterface(gomock.NewController(t))
		verifier := NewActiveUserVerifier(repo)

		repo.EXPECT().FindById(gomock.Any(), uid).Return(&entity.User{ID: uid}, nil)
		active, err := verifier.IsActiveUser(context.Background(), uid)
		require.NoError(t, err)
		require.True(t, active)

		// TTL 経過後は退会済み(見つからない)として無効になる
		overrideTimeNow(t, now.Add(activeUserCacheTTL+time.Second))
		repo.EXPECT().FindById(gomock.Any(), uid).Return(nil, apperror.ErrRecordNotFound)
		active, err = verifier.IsActiveUser(context.Background(), uid)
		require.NoError(t, err)
		require.False(t, active)
	})

	// 否定の結果は保持しない(登録直後のリクエストが古い結果で弾かれないように)
	t.Run("正常系_無効の結果は保持せず次回もDBで確認する", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockUserInterface(gomock.NewController(t))
		verifier := NewActiveUserVerifier(repo)

		repo.EXPECT().FindById(gomock.Any(), uid).Return(nil, apperror.ErrRecordNotFound)
		active, err := verifier.IsActiveUser(context.Background(), uid)
		require.NoError(t, err)
		require.False(t, active)

		// 登録された直後(同じ時刻)でも有効になる
		repo.EXPECT().FindById(gomock.Any(), uid).Return(&entity.User{ID: uid}, nil)
		active, err = verifier.IsActiveUser(context.Background(), uid)
		require.NoError(t, err)
		require.True(t, active)
	})
}
