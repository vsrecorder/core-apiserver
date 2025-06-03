package auth

import (
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
	secretKey := "JrScU7NuTAAp4mjrXhKZlgYoFwXrHhEyPVSpYLukOZg="
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken("zor5SLfEfwfZ90yRVXzlxBEFARy2", secretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := RequiredAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", uid)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		secretKey := "FBIN08bOcuvO2X+S6p1yD9lRxOdby+YjUQgmFQsoQ1c="

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken("zor5SLfEfwfZ90yRVXzlxBEFARy2", secretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := RequiredAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, "", uid)
	})

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
	secretKey := "JrScU7NuTAAp4mjrXhKZlgYoFwXrHhEyPVSpYLukOZg="
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken("zor5SLfEfwfZ90yRVXzlxBEFARy2", secretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := OptionalAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "zor5SLfEfwfZ90yRVXzlxBEFARy2", uid)
	})

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

	t.Run("異常系_#01", func(t *testing.T) {
		wrongSecretKey := "FBIN08bOcuvO2X+S6p1yD9lRxOdby+YjUQgmFQsoQ1c="

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateToken("zor5SLfEfwfZ90yRVXzlxBEFARy2", wrongSecretKey)
		require.NoError(t, err)

		req.Header.Add("Authorization", "Bearer "+token)

		ginContext.Request = req

		middleware := OptionalAuthenticationMiddleware()
		middleware(ginContext)

		uid := helper.GetUID(ginContext)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, "", uid)
	})

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
