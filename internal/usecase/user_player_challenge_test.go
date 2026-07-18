package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func TestUserPlayerChallenge(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	playerId := "1234567890123456"
	avatarImageURL := "https://example.com/avatar.png"

	t.Run("正常系_発行したトークンを検証すると発行時の内容が取り出せる", func(t *testing.T) {
		secret, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secret)

		token, expiresAt, err := signUserPlayerChallenge(uid, playerId, avatarImageURL)
		require.NoError(t, err)
		require.NotEmpty(t, token)
		// 有効期限は発行から約10分後
		require.WithinDuration(t, time.Now().Add(userPlayerChallengeTTL), expiresAt, time.Minute)

		claims, err := parseUserPlayerChallenge(token)
		require.NoError(t, err)
		require.Equal(t, uid, claims.UID)
		require.Equal(t, playerId, claims.PlayerId)
		require.Equal(t, avatarImageURL, claims.ChallengeAvatarImageURL)
	})

	t.Run("異常系_シークレット未設定なら発行に失敗する", func(t *testing.T) {
		t.Setenv("VSRECORDER_JWT_SECRET", "")

		_, _, err := signUserPlayerChallenge(uid, playerId, avatarImageURL)

		require.Error(t, err)
	})

	t.Run("異常系_シークレット未設定なら検証はErrInvalidChallengeを返す", func(t *testing.T) {
		t.Setenv("VSRECORDER_JWT_SECRET", "")

		_, err := parseUserPlayerChallenge("whatever")

		require.Equal(t, apperror.ErrInvalidChallenge, err)
	})

	t.Run("異常系_改ざんされたトークンはErrInvalidChallengeを返す", func(t *testing.T) {
		secret, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secret)

		token, _, err := signUserPlayerChallenge(uid, playerId, avatarImageURL)
		require.NoError(t, err)

		_, err = parseUserPlayerChallenge(token + "tampered")

		require.Equal(t, apperror.ErrInvalidChallenge, err)
	})

	t.Run("異常系_異なるシークレットで署名されたトークンはErrInvalidChallengeを返す", func(t *testing.T) {
		otherSecret, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", otherSecret)

		token, _, err := signUserPlayerChallenge(uid, playerId, avatarImageURL)
		require.NoError(t, err)

		// 検証時には別のシークレットが設定されている
		secret, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secret)

		_, err = parseUserPlayerChallenge(token)

		require.Equal(t, apperror.ErrInvalidChallenge, err)
	})
}
