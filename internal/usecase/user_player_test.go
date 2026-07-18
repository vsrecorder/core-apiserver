package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
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

	t.Run("Verify", func(t *testing.T) {
		t.Run("正常系_実在確認の結果と現在と異なるアバターのチャレンジを返す", func(t *testing.T) {
			secret, err := testutil.GenerateJWTSecret()
			require.NoError(t, err)
			t.Setenv("VSRECORDER_JWT_SECRET", secret)

			overridePlayerAccountAPI(t, func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, `{"code":200,"player":{
					"player_id":"1234567890123456",
					"nickname":"ニックネーム",
					"avatar_image":"https://example.com/current-avatar.png"
				}}`)
			})

			_, _, usecase := setup4UserPlayerUsecase(t)

			ret, err := usecase.Verify(context.Background(), uid, playerId)

			require.NoError(t, err)
			require.Equal(t, playerId, ret.Account.PlayerId)
			// チャレンジには現在のアバターと異なる画像が提示される
			// (stubPokemonAvatarRepositoryが返すother-avatar.png)
			require.Equal(t, "https://example.com/other-avatar.png", ret.Challenge.ChallengeAvatarImageURL)
			require.NotEmpty(t, ret.Challenge.Token)

			// 発行されたトークンは発行時の内容で検証できる
			claims, err := parseUserPlayerChallenge(ret.Challenge.Token)
			require.NoError(t, err)
			require.Equal(t, uid, claims.UID)
			require.Equal(t, playerId, claims.PlayerId)
		})

		t.Run("異常系_プレイヤーが実在しなければErrRecordNotFoundを返す", func(t *testing.T) {
			overridePlayerAccountAPI(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"code":404}`)
			})

			_, _, usecase := setup4UserPlayerUsecase(t)

			ret, err := usecase.Verify(context.Background(), uid, playerId)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
		})
	})

	t.Run("Create", func(t *testing.T) {
		// signChallengeAndServeAvatar はチャレンジトークンを発行し、プレイヤーズクラブAPIが
		// avatarImageを返すよう設定する(=アバター変更済みの状態を再現する)。
		signChallengeAndServeAvatar := func(t *testing.T, avatarImage string) string {
			t.Helper()

			secret, err := testutil.GenerateJWTSecret()
			require.NoError(t, err)
			t.Setenv("VSRECORDER_JWT_SECRET", secret)

			token, _, err := signUserPlayerChallenge(uid, playerId, "https://example.com/other-avatar.png")
			require.NoError(t, err)

			overridePlayerAccountAPI(t, func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprintf(w, `{"code":200,"player":{"player_id":%q,"avatar_image":%q}}`, playerId, avatarImage)
			})

			return token
		}

		t.Run("正常系_アバター変更を確認できたら紐付けを作成する", func(t *testing.T) {
			token := signChallengeAndServeAvatar(t, "https://example.com/other-avatar.png")

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)
			mockRepository.EXPECT().ExistsActiveByPlayerId(context.Background(), playerId).Return(false, nil)
			mockRepository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.NoError(t, err)
			require.NotEmpty(t, ret.ID)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, playerId, ret.PlayerId)
		})

		t.Run("正常系_1ヶ月経過後の変更は旧紐付けを削除してから作成する", func(t *testing.T) {
			token := signChallengeAndServeAvatar(t, "https://example.com/other-avatar.png")

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			// 2ヶ月前の紐付け(別のplayer_id)が存在する
			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local().AddDate(0, -2, 0), uid, "9999999999999999")

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)
			mockRepository.EXPECT().ExistsActiveByPlayerId(context.Background(), playerId).Return(false, nil)
			mockRepository.EXPECT().Delete(gomock.Any(), existing.ID).Return(nil)
			mockRepository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.NoError(t, err)
			require.Equal(t, playerId, ret.PlayerId)
		})

		t.Run("正常系_同じプレイヤーIDなら変更不要として既存の紐付けを返す", func(t *testing.T) {
			token := signChallengeAndServeAvatar(t, "https://example.com/other-avatar.png")

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local(), uid, playerId)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.NoError(t, err)
			require.Equal(t, existing, ret)
		})

		t.Run("異常系_アバターが変更されていなければErrOwnershipNotVerifiedを返す", func(t *testing.T) {
			// チャレンジで指定した画像と異なるアバターのまま
			token := signChallengeAndServeAvatar(t, "https://example.com/current-avatar.png")

			_, _, usecase := setup4UserPlayerUsecase(t)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrOwnershipNotVerified)
			require.Nil(t, ret)
		})

		t.Run("異常系_紐付けから1ヶ月未満の変更はErrLockedを返す", func(t *testing.T) {
			token := signChallengeAndServeAvatar(t, "https://example.com/other-avatar.png")

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			// 直近に別のplayer_idを紐付けたばかり
			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local(), uid, "9999999999999999")

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrLocked)
			require.Nil(t, ret)
		})

		t.Run("異常系_別ユーザに紐付け済みのプレイヤーIDはErrAlreadyExistsを返す", func(t *testing.T) {
			token := signChallengeAndServeAvatar(t, "https://example.com/other-avatar.png")

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)
			mockRepository.EXPECT().ExistsActiveByPlayerId(context.Background(), playerId).Return(true, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrAlreadyExists)
			require.Nil(t, ret)
		})

		t.Run("異常系_不正なチャレンジトークンはErrInvalidChallengeを返す", func(t *testing.T) {
			secret, err := testutil.GenerateJWTSecret()
			require.NoError(t, err)
			t.Setenv("VSRECORDER_JWT_SECRET", secret)

			_, _, usecase := setup4UserPlayerUsecase(t)

			param := NewUserPlayerCreateParam(uid, playerId, "invalid-token")

			ret, err := usecase.Create(context.Background(), param)

			require.ErrorIs(t, err, apperror.ErrInvalidChallenge)
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

			require.ErrorIs(t, err, apperror.ErrInvalidChallenge)
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

			require.ErrorIs(t, err, apperror.ErrInvalidChallenge)
			require.Nil(t, ret)
		})
	})
}
