package auth

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

const (
	TokenLifetimeSecond = time.Duration(15) * time.Second
)

type VSRClaims struct {
	jwt.RegisteredClaims
	UID string `json:"uid"`
}

func parseToken(tokenString string, secretKey string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &VSRClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func RequiredAuthenticationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		secretKey := os.Getenv("VSRECORDER_JWT_SECRET")

		header := http.Header{}
		header.Add("Authorization", ctx.GetHeader("Authorization"))

		tokenString := strings.TrimPrefix(header.Get("Authorization"), "Bearer ")

		token, err := parseToken(tokenString, secretKey)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			ctx.Abort()
			return
		}

		claims := token.Claims.(*VSRClaims)

		if claims.UID == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			ctx.Abort()
			return
		}

		helper.SetUID(ctx, claims.UID)
	}
}

func OptionalAuthenticationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		secretKey := os.Getenv("VSRECORDER_JWT_SECRET")

		header := http.Header{}
		header.Add("Authorization", ctx.GetHeader("Authorization"))

		if header.Get("Authorization") == "" {
			helper.SetUID(ctx, "")
			return
		}

		tokenString := strings.TrimPrefix(header.Get("Authorization"), "Bearer ")

		token, err := parseToken(tokenString, secretKey)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			ctx.Abort()
			return
		}

		claims := token.Claims.(*VSRClaims)

		if claims.UID == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			ctx.Abort()
			return
		}

		helper.SetUID(ctx, claims.UID)
	}
}
