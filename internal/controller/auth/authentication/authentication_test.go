package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func generateToken(uid string, secretKey string) (string, error) {
	return generateTokenWithIssuer(uid, secretKey, ExpectedIssuer)
}

func generateTokenWithIssuer(uid string, secretKey string, issuer string) (string, error) {
	return testutil.GenerateJWT(uid, secretKey, issuer)
}

func GenerateJWTSecret() (string, error) {
	return testutil.GenerateJWTSecret()
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
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 正常系のテスト
	t.Run("正常系_有効なトークンならUIDを設定して通過する", func(t *testing.T) {
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
	t.Run("異常系_署名キーが異なるトークンは401を返す", func(t *testing.T) {
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
	t.Run("異常系_トークンがなければ401を返す", func(t *testing.T) {
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
	t.Run("異常系_UIDが空のトークンは401を返す", func(t *testing.T) {
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

	// JWTトークンのissuerが不正な場合のテスト
	t.Run("異常系_issuerが不正なトークンは401を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateTokenWithIssuer(userId, secretKey, "unexpected-issuer")
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
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 正常系のテスト
	t.Run("正常系_有効なトークンならUIDを設定して通過する", func(t *testing.T) {
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
	t.Run("正常系_トークンなしでもUID空のまま通過する", func(t *testing.T) {
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
	t.Run("異常系_署名キーが異なるトークンは401を返す", func(t *testing.T) {
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
	t.Run("異常系_UIDが空のトークンは401を返す", func(t *testing.T) {
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

	// JWTトークンのissuerが不正な場合のテスト
	t.Run("異常系_issuerが不正なトークンは401を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		token, err := generateTokenWithIssuer(userId, secretKey, "unexpected-issuer")
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
