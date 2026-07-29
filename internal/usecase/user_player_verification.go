package usecase

import (
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

// userPlayerVerificationIssuer は検証済みトークンの発行者。
//
// プレイヤーズクラブでの実在確認とアバター変更による所有権確認は webapp(BFF)が行い、
// その結果を署名付きトークンとしてこのAPIサーバへ渡す。webapp とは
// VSRECORDER_JWT_SECRET を共有しているため、署名が検証できることをもって
// 「webapp が確認した」と扱える。
//
// iss は認証用トークン(vsrecorder-webapp)とは必ず別にする。同じにすると、
// 用途の異なるトークンが互いに通用してしまう。
const userPlayerVerificationIssuer = "vsrecorder-webapp-user-player-verification"

type userPlayerVerificationClaims struct {
	jwt.RegisteredClaims
	UID      string `json:"uid"`
	PlayerId string `json:"player_id"`
}

// parseUserPlayerVerification は検証済みトークンを検証し、改ざん・期限切れであれば
// apperror.ErrInvalidVerification を返す。
func parseUserPlayerVerification(tokenString string) (*userPlayerVerificationClaims, error) {
	secret := os.Getenv("VSRECORDER_JWT_SECRET")

	// 空鍵で検証すると、空鍵([]byte(""))で署名された偽造トークンを正当なものとして
	// 受け入れてしまう。所有権確認が丸ごと破れるため、必ず失敗させる。
	if secret == "" {
		return nil, apperror.ErrInvalidVerification
	}

	token, err := jwt.ParseWithClaims(tokenString, &userPlayerVerificationClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(secret), nil
	},
		jwt.WithIssuer(userPlayerVerificationIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, apperror.ErrInvalidVerification
	}

	claims, ok := token.Claims.(*userPlayerVerificationClaims)
	if !ok || !token.Valid {
		return nil, apperror.ErrInvalidVerification
	}

	return claims, nil
}
