package authentication

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func generateToken(uid string, secretKey string) (string, error) {
	claims := jwt.MapClaims{
		"uid": uid,
		"exp": time.Now().Add(TokenLifetimeSecond).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GenerateJWTSecret() (string, error) {
	key := make([]byte, 32) // 256bit
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

func TestAuthenticationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"RequiredAuthenticationMiddleware": test_RequiredAuthenticationMiddleware,
		"OptionalAuthenticationMiddleware": test_OptionalAuthenticationMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_RequiredAuthenticationMiddleware(t *testing.T) {
	// JWTのシークレットキーを生成する
	secretKey, err := GenerateJWTSecret()
	require.NoError(t, err)

	// 環境変数にJWTのシークレットキーをセットする
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 正常系のテスト
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken(userId, secretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := RequiredAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, userId, uid)
	})

	// JWTトークンの署名に使用するシークレットキーが間違っている場合のテスト
	t.Run("異常系_#01", func(t *testing.T) {
		wrongSecretKey, err := GenerateJWTSecret()
		require.NoError(t, err)

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken(userId, wrongSecretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := RequiredAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		// uidはセットされていないはず
		require.Equal(t, "", uid)
	})

	// JWTトークンがない場合のテスト
	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add("Authorization", "")

		ginContext.Request = req

		middleware := RequiredAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, "", uid)
	})

	// userIdが空のJWTトークンの場合のテスト
	t.Run("異常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken("", secretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := RequiredAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, "", uid)
	})
}

func test_OptionalAuthenticationMiddleware(t *testing.T) {
	// JWTのシークレットキーを生成する
	secretKey, err := GenerateJWTSecret()
	require.NoError(t, err)

	// 環境変数にJWTのシークレットキーをセットする
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 正常系のテスト
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken(userId, secretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := OptionalAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, userId, uid)
	})

	// 正常系のテスト(JWTトークンがない場合)
	t.Run("正常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add("Authorization", "")

		ginContext.Request = req

		middleware := OptionalAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "", uid)
	})

	// JWTトークンの署名に使用するシークレットキーが間違っている場合のテスト
	t.Run("異常系_#01", func(t *testing.T) {
		wrongSecretKey, err := GenerateJWTSecret()
		require.NoError(t, err)

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken(userId, wrongSecretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := OptionalAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, "", uid)
	})

	// userIdが空のJWTトークンの場合のテスト
	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken("", secretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := OptionalAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, "", uid)
	})
}
