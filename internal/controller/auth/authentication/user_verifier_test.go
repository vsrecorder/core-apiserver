package authentication

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

// setUserVerifierForTest は uid の状態を答える関数を差し替え、テスト終了時に未設定へ戻す。
func setUserVerifierForTest(t *testing.T, f func(ctx context.Context, uid string) (bool, error)) {
	t.Helper()

	SetUserVerifier(UserVerifierFunc(f))
	t.Cleanup(func() { SetUserVerifier(nil) })
}

// allowAllUsers は全ての uid を有効なユーザーとして通す(署名検証だけを見たいテスト用)。
func allowAllUsers(t *testing.T) {
	t.Helper()

	setUserVerifierForTest(t, func(context.Context, string) (bool, error) { return true, nil })
}

func newAuthenticatedContext(t *testing.T, uid string, secretKey string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	token, err := generateToken(uid, secretKey)
	require.NoError(t, err)
	req.Header.Add("Authorization", "Bearer "+token)

	ctx.Request = req

	return ctx, w
}

// 署名が正しいトークンでも、uid が未登録・退会済みなら通さない。
// 退会後に Firebase 側のアカウントが残っていても、そのトークンで書き込めないようにするため。
func TestAuthenticationMiddleware_UserVerifier(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey, err := GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	for name, middleware := range map[string]func() gin.HandlerFunc{
		"RequiredAuthenticationMiddleware": RequiredAuthenticationMiddleware,
		"OptionalAuthenticationMiddleware": OptionalAuthenticationMiddleware,
	} {
		t.Run(name, func(t *testing.T) {
			t.Run("異常系_未登録または退会済みのuidは401を返す", func(t *testing.T) {
				var askedUid string
				setUserVerifierForTest(t, func(_ context.Context, u string) (bool, error) {
					askedUid = u
					return false, nil
				})

				ctx, w := newAuthenticatedContext(t, uid, secretKey)
				middleware()(ctx)

				require.Equal(t, http.StatusUnauthorized, w.Code)
				require.True(t, ctx.IsAborted())
				require.Equal(t, uid, askedUid)
			})

			t.Run("異常系_確認に失敗したら500を返す", func(t *testing.T) {
				setUserVerifierForTest(t, func(context.Context, string) (bool, error) {
					return false, errors.New("db down")
				})

				ctx, w := newAuthenticatedContext(t, uid, secretKey)
				middleware()(ctx)

				require.Equal(t, http.StatusInternalServerError, w.Code)
				require.True(t, ctx.IsAborted())
			})

			// 検査を黙って省略する fail open にしない
			t.Run("異常系_検証先が未設定なら500を返す", func(t *testing.T) {
				SetUserVerifier(nil)

				ctx, w := newAuthenticatedContext(t, uid, secretKey)
				middleware()(ctx)

				require.Equal(t, http.StatusInternalServerError, w.Code)
				require.True(t, ctx.IsAborted())
			})

			t.Run("正常系_有効なユーザーなら通過する", func(t *testing.T) {
				allowAllUsers(t)

				ctx, w := newAuthenticatedContext(t, uid, secretKey)
				middleware()(ctx)

				require.Equal(t, http.StatusOK, w.Code)
				require.False(t, ctx.IsAborted())
				require.Equal(t, uid, helper.GetUID(ctx))
			})
		})
	}

	// 登録前の uid は users に無いので、登録用の認証は uid の状態を見ない
	t.Run("RegistrationAuthenticationMiddleware", func(t *testing.T) {
		t.Run("正常系_未登録のuidでも署名が正しければ通過する", func(t *testing.T) {
			setUserVerifierForTest(t, func(context.Context, string) (bool, error) {
				t.Fatal("registration must not consult the user verifier")
				return false, nil
			})

			ctx, w := newAuthenticatedContext(t, uid, secretKey)
			RegistrationAuthenticationMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uid, helper.GetUID(ctx))
		})

		t.Run("異常系_署名が不正なら401を返す", func(t *testing.T) {
			otherKey, err := GenerateJWTSecret()
			require.NoError(t, err)

			ctx, w := newAuthenticatedContext(t, uid, otherKey)
			RegistrationAuthenticationMiddleware()(ctx)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})
}
