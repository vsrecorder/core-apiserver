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
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

// stubPokemonAvatarRepository はアバター取得のスタブ。
// mock_repositoryにPokemonAvatar用のモックが存在しないため手書きする。
type stubPokemonAvatarRepository struct{}

func (stubPokemonAvatarRepository) FindRandomExcludingImageURL(ctx context.Context, imageURL string) (*entity.PokemonAvatar, error) {
	return &entity.PokemonAvatar{ImageURL: "https://example.com/other-avatar.png"}, nil
}

func setup4UserPlayerUsecase(t *testing.T) (
	*mock_repository.MockUserPlayerInterface,
	*mock_repository.MockPlayerRankingInterface,
	UserPlayerInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserPlayerInterface(mockCtrl)
	mockAvatarRepository := stubPokemonAvatarRepository{}
	mockPlayerRankingRepository := mock_repository.NewMockPlayerRankingInterface(mockCtrl)
	mockTransactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	mockTransactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()

	usecase := NewUserPlayer(mockRepository, mockAvatarRepository, mockPlayerRankingRepository, mockTransactionManager)

	return mockRepository, mockPlayerRankingRepository, usecase
}

func TestUserPlayerUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	playerId := "1234567890123456"

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_指定ユーザの紐付けを返す", func(t *testing.T) {
			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			userPlayer := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN2", time.Now().Local(), uid, playerId)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(userPlayer, nil)

			ret, err := usecase.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, playerId, ret.PlayerId)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.FindByUserId(context.Background(), uid)

			require.Equal(t, apperror.ErrRecordNotFound, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindLatestPlayerRanking", func(t *testing.T) {
		t.Run("正常系_最新のランキング情報を返す", func(t *testing.T) {
			_, mockPlayerRankingRepository, usecase := setup4UserPlayerUsecase(t)

			ranking := &entity.PlayerRanking{PlayerId: playerId}

			mockPlayerRankingRepository.EXPECT().FindLatestByPlayerId(context.Background(), playerId).Return(ranking, nil)

			ret, err := usecase.FindLatestPlayerRanking(context.Background(), playerId)

			require.NoError(t, err)
			require.Equal(t, playerId, ret.PlayerId)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			_, mockPlayerRankingRepository, usecase := setup4UserPlayerUsecase(t)

			mockPlayerRankingRepository.EXPECT().FindLatestByPlayerId(context.Background(), playerId).Return(nil, errors.New(""))

			ret, err := usecase.FindLatestPlayerRanking(context.Background(), playerId)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("異常系_不正なチャレンジトークンはErrInvalidChallengeを返す", func(t *testing.T) {
			secret, err := testutil.GenerateJWTSecret()
			require.NoError(t, err)
			t.Setenv("VSRECORDER_JWT_SECRET", secret)

			_, _, usecase := setup4UserPlayerUsecase(t)

			param := NewUserPlayerCreateParam(uid, playerId, "invalid-token")

			ret, err := usecase.Create(context.Background(), param)

			require.Equal(t, apperror.ErrInvalidChallenge, err)
			require.Nil(t, ret)
		})

		t.Run("異常系_別ユーザ宛に発行されたチャレンジはErrInvalidChallengeを返す", func(t *testing.T) {
			secret, err := testutil.GenerateJWTSecret()
			require.NoError(t, err)
			t.Setenv("VSRECORDER_JWT_SECRET", secret)

			_, _, usecase := setup4UserPlayerUsecase(t)

			// 別のユーザ向けに発行されたトークンを使う
			token, _, err := signUserPlayerChallenge("other-user", playerId, "https://example.com/avatar.png")
			require.NoError(t, err)

			param := NewUserPlayerCreateParam(uid, playerId, token)

			ret, err := usecase.Create(context.Background(), param)

			require.Equal(t, apperror.ErrInvalidChallenge, err)
			require.Nil(t, ret)
		})

		t.Run("異常系_別のプレイヤーID宛のチャレンジはErrInvalidChallengeを返す", func(t *testing.T) {
			secret, err := testutil.GenerateJWTSecret()
			require.NoError(t, err)
			t.Setenv("VSRECORDER_JWT_SECRET", secret)

			_, _, usecase := setup4UserPlayerUsecase(t)

			token, _, err := signUserPlayerChallenge(uid, "9999999999999999", "https://example.com/avatar.png")
			require.NoError(t, err)

			param := NewUserPlayerCreateParam(uid, playerId, token)

			ret, err := usecase.Create(context.Background(), param)

			require.Equal(t, apperror.ErrInvalidChallenge, err)
			require.Nil(t, ret)
		})
	})
}
