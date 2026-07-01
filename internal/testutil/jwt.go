package testutil

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWTSecret はテスト用のランダムなJWT署名鍵を生成する。
func GenerateJWTSecret() (string, error) {
	key := make([]byte, 32) // 256bit
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

// GenerateJWT はテスト用にHS256で署名されたJWTを生成する。
func GenerateJWT(uid string, secretKey string, issuer string) (string, error) {
	claims := jwt.MapClaims{
		"uid": uid,
		"iss": issuer,
		"exp": time.Now().Add(15 * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
