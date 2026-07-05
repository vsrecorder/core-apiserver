package usecase

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

const (
	userPlayerChallengeIssuer = "vsrecorder-core-apiserver"
	userPlayerChallengeTTL    = 10 * time.Minute
)

// OwnershipChallenge はプレイヤーID所有権確認(アバター変更チャレンジ)の内容を表す。
// ChallengeAvatar* は「これに変更してください」と提示するアバター、Token はその内容を
// 署名して埋め込んだ、次の登録リクエストで検証するためのトークン。
type OwnershipChallenge struct {
	Token                   string
	ChallengeAvatarID       int
	ChallengeAvatarTitle    string
	ChallengeAvatarImageURL string
	ChallengeAvatarDetail   string
	ExpiresAt               time.Time
}

type userPlayerChallengeClaims struct {
	jwt.RegisteredClaims
	UID                     string `json:"uid"`
	PlayerId                string `json:"player_id"`
	ChallengeAvatarImageURL string `json:"challenge_avatar_image_url"`
}

// signUserPlayerChallenge は「このユーザーが、このplayer_idについて、この
// アバター画像への変更を求められた」という事実を署名済みトークンとして発行する。
func signUserPlayerChallenge(uid string, playerId string, challengeAvatarImageURL string) (string, time.Time, error) {
	secret := os.Getenv("VSRECORDER_JWT_SECRET")
	now := time.Now()
	expiresAt := now.Add(userPlayerChallengeTTL)

	claims := userPlayerChallengeClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    userPlayerChallengeIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UID:                     uid,
		PlayerId:                playerId,
		ChallengeAvatarImageURL: challengeAvatarImageURL,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

// parseUserPlayerChallenge はトークンを検証し、改ざん・期限切れであれば
// apperror.ErrInvalidChallenge を返す。
func parseUserPlayerChallenge(tokenString string) (*userPlayerChallengeClaims, error) {
	secret := os.Getenv("VSRECORDER_JWT_SECRET")

	token, err := jwt.ParseWithClaims(tokenString, &userPlayerChallengeClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(secret), nil
	}, jwt.WithIssuer(userPlayerChallengeIssuer))
	if err != nil {
		return nil, apperror.ErrInvalidChallenge
	}

	claims, ok := token.Claims.(*userPlayerChallengeClaims)
	if !ok || !token.Valid {
		return nil, apperror.ErrInvalidChallenge
	}

	return claims, nil
}
