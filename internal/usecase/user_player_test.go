package usecase

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

// stubPokemonAvatarRepository はアバター取得のスタブ。
// mock_repositoryにPokemonAvatar用のモックが存在しないため手書きする。
type stubPokemonAvatarRepository struct {
	err error
}

func (s stubPokemonAvatarRepository) FindRandomExcludingImageURL(ctx context.Context, imageURL string) (*entity.PokemonAvatar, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &entity.PokemonAvatar{ImageURL: "https://example.com/other-avatar.png"}, nil
}

func setup4UserPlayerUsecase(t *testing.T) (
	*mock_repository.MockUserPlayerInterface,
	*mock_repository.MockPlayerRankingInterface,
	UserPlayerInterface,
) {
	return setup4UserPlayerUsecaseWithAvatar(t, stubPokemonAvatarRepository{})
}

func setup4UserPlayerUsecaseWithAvatar(t *testing.T, avatarRepository stubPokemonAvatarRepository) (
	*mock_repository.MockUserPlayerInterface,
	*mock_repository.MockPlayerRankingInterface,
	UserPlayerInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserPlayerInterface(mockCtrl)
	mockPlayerRankingRepository := mock_repository.NewMockPlayerRankingInterface(mockCtrl)
	mockTransactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	mockTransactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()

	usecase := NewUserPlayer(mockRepository, avatarRepository, mockPlayerRankingRepository, mockTransactionManager)

	return mockRepository, mockPlayerRankingRepository, usecase
}

// signVerificationForTest は webapp が発行する検証済みトークンを模して署名する。
// issuer を差し替えられるようにし、用途の異なるトークンが通らないことも検証できるようにする。
func signVerificationForTest(t *testing.T, issuer string, uid string, playerId string, expiresAt time.Time) string {
	t.Helper()

	claims := userPlayerVerificationClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UID:      uid,
		PlayerId: playerId,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(os.Getenv("VSRECORDER_JWT_SECRET")))
	require.NoError(t, err)

	return token
}

func TestUserPlayerUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	playerId := "1234567890123456"

	// setJWTSecret は検証済みトークンの署名・検証に使う共有鍵を設定する。
	setJWTSecret := func(t *testing.T) {
		t.Helper()

		secret, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secret)
	}

	// validToken は正当な検証済みトークンを返す。
	validToken := func(t *testing.T) string {
		t.Helper()

		setJWTSecret(t)

		return signVerificationForTest(t, userPlayerVerificationIssuer, uid, playerId, time.Now().Add(time.Minute))
	}

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

	t.Run("IssueChallengeAvatar", func(t *testing.T) {
		t.Run("正常系_現在と異なるアバターを払い出す", func(t *testing.T) {
			_, _, usecase := setup4UserPlayerUsecase(t)

			ret, err := usecase.IssueChallengeAvatar(context.Background(), "https://example.com/current-avatar.png")

			require.NoError(t, err)
			require.Equal(t, "https://example.com/other-avatar.png", ret.ImageURL)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			_, _, usecase := setup4UserPlayerUsecaseWithAvatar(t, stubPokemonAvatarRepository{err: apperror.ErrRecordNotFound})

			ret, err := usecase.IssueChallengeAvatar(context.Background(), "https://example.com/current-avatar.png")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_検証済みトークンが正しければ紐付けを作成する", func(t *testing.T) {
			token := validToken(t)

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
			token := validToken(t)

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
			token := validToken(t)

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local(), uid, playerId)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.NoError(t, err)
			require.Equal(t, existing, ret)
		})

		t.Run("異常系_紐付けから1ヶ月未満の変更はErrLockedを返す", func(t *testing.T) {
			token := validToken(t)

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			// 直近に別のplayer_idを紐付けたばかり
			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local(), uid, "9999999999999999")

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrLocked)
			require.Nil(t, ret)
		})

		t.Run("異常系_別ユーザに紐付け済みのプレイヤーIDはErrAlreadyExistsを返す", func(t *testing.T) {
			token := validToken(t)

			mockRepository, _, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)
			mockRepository.EXPECT().ExistsActiveByPlayerId(context.Background(), playerId).Return(true, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrAlreadyExists)
			require.Nil(t, ret)
		})

		t.Run("異常系_不正な検証済みトークンはErrInvalidVerificationを返す", func(t *testing.T) {
			setJWTSecret(t)

			_, _, usecase := setup4UserPlayerUsecase(t)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, "invalid-token"))

			require.ErrorIs(t, err, apperror.ErrInvalidVerification)
			require.Nil(t, ret)
		})

		t.Run("異常系_別ユーザ宛の検証済みトークンはErrInvalidVerificationを返す", func(t *testing.T) {
			setJWTSecret(t)

			_, _, usecase := setup4UserPlayerUsecase(t)

			token := signVerificationForTest(t, userPlayerVerificationIssuer, "other-user", playerId, time.Now().Add(time.Minute))

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrInvalidVerification)
			require.Nil(t, ret)
		})

		t.Run("異常系_別のプレイヤーID宛の検証済みトークンはErrInvalidVerificationを返す", func(t *testing.T) {
			setJWTSecret(t)

			_, _, usecase := setup4UserPlayerUsecase(t)

			token := signVerificationForTest(t, userPlayerVerificationIssuer, uid, "9999999999999999", time.Now().Add(time.Minute))

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrInvalidVerification)
			require.Nil(t, ret)
		})

		t.Run("異常系_期限切れの検証済みトークンはErrInvalidVerificationを返す", func(t *testing.T) {
			setJWTSecret(t)

			_, _, usecase := setup4UserPlayerUsecase(t)

			token := signVerificationForTest(t, userPlayerVerificationIssuer, uid, playerId, time.Now().Add(-time.Minute))

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrInvalidVerification)
			require.Nil(t, ret)
		})

		// 認証用トークンは同じ鍵・同じuidで署名されているため、issを区別しないと
		// 所有権を確認していないのに紐付けが通ってしまう。
		t.Run("異常系_認証用トークンを流用してもErrInvalidVerificationを返す", func(t *testing.T) {
			setJWTSecret(t)

			_, _, usecase := setup4UserPlayerUsecase(t)

			token := signVerificationForTest(t, "vsrecorder-webapp", uid, playerId, time.Now().Add(time.Minute))

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrInvalidVerification)
			require.Nil(t, ret)
		})

		t.Run("異常系_共有鍵が未設定ならErrInvalidVerificationを返す", func(t *testing.T) {
			setJWTSecret(t)

			_, _, usecase := setup4UserPlayerUsecase(t)

			token := signVerificationForTest(t, userPlayerVerificationIssuer, uid, playerId, time.Now().Add(time.Minute))

			// 署名後に鍵を空にする(空鍵で署名された偽造トークンを受け入れないこと)
			t.Setenv("VSRECORDER_JWT_SECRET", "")

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId, token))

			require.ErrorIs(t, err, apperror.ErrInvalidVerification)
			require.Nil(t, ret)
		})
	})
}
