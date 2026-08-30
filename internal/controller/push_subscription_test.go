package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestPushSubscriptionController(t *testing.T) (
	*PushSubscription,
	*mock_usecase.MockPushSubscriptionInterface,
	string,
) {
	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockPushSubscriptionInterface(mockCtrl)

	r := gin.Default()
	c := NewPushSubscription(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func newPushSubscriptionRequest(t *testing.T, method string, body string, uid string, secretKey string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, UsersPath+PushSubscriptionsPath, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if uid != "" {
		setJWTAuthHeader(t, req, uid, secretKey)
	}

	return req
}

func TestPushSubscriptionController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	endpoint := "https://fcm.googleapis.com/fcm/send/abc"

	t.Run("Subscribe", func(t *testing.T) {
		t.Run("正常系_購読をユースケースへ渡して204を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushSubscriptionController(t)

			mockUsecase.EXPECT().Subscribe(gomock.Any(), uid, endpoint, "p256dh-key", "auth-key", entity.PushPlatformAndroid).Return(nil)

			body := `{"endpoint":"` + endpoint + `","keys":{"p256dh":"p256dh-key","auth":"auth-key"},"platform":"android"}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "POST", body, uid, secretKey))

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("正常系_未知のplatformは空文字に丸めて受け付ける", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushSubscriptionController(t)

			mockUsecase.EXPECT().Subscribe(gomock.Any(), uid, endpoint, "p256dh-key", "auth-key", "").Return(nil)

			body := `{"endpoint":"` + endpoint + `","keys":{"p256dh":"p256dh-key","auth":"auth-key"},"platform":"smart-fridge"}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "POST", body, uid, secretKey))

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_購読数が上限ならErrTooManyPushSubscriptionsから409を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushSubscriptionController(t)

			mockUsecase.EXPECT().Subscribe(gomock.Any(), uid, endpoint, "p256dh-key", "auth-key", "").Return(apperror.ErrTooManyPushSubscriptions)

			body := `{"endpoint":"` + endpoint + `","keys":{"p256dh":"p256dh-key","auth":"auth-key"}}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "POST", body, uid, secretKey))

			require.Equal(t, http.StatusConflict, w.Code)
		})

		t.Run("異常系_https以外のendpointは400", func(t *testing.T) {
			c, _, secretKey := setup4TestPushSubscriptionController(t)

			body := `{"endpoint":"http://example.com/push","keys":{"p256dh":"p256dh-key","auth":"auth-key"}}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "POST", body, uid, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_IPアドレス直指定やlocalhostのendpointは400", func(t *testing.T) {
			for _, endpoint := range []string{
				"https://127.0.0.1/push",
				"https://[::1]/push",
				"https://localhost/push",
				"https://api.local/push",
				"https://push-service/no-dot",
				"https://user:pass@fcm.googleapis.com/fcm/send/abc",
			} {
				c, _, secretKey := setup4TestPushSubscriptionController(t)

				body := `{"endpoint":"` + endpoint + `","keys":{"p256dh":"p256dh-key","auth":"auth-key"}}`
				w := httptest.NewRecorder()
				c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "POST", body, uid, secretKey))

				require.Equal(t, http.StatusBadRequest, w.Code, endpoint)
			}
		})

		t.Run("異常系_鍵が無ければ400", func(t *testing.T) {
			c, _, secretKey := setup4TestPushSubscriptionController(t)

			body := `{"endpoint":"` + endpoint + `","keys":{"p256dh":"","auth":"auth-key"}}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "POST", body, uid, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_未認証は401", func(t *testing.T) {
			c, _, secretKey := setup4TestPushSubscriptionController(t)

			body := `{"endpoint":"` + endpoint + `","keys":{"p256dh":"p256dh-key","auth":"auth-key"}}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "POST", body, "", secretKey))

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		t.Run("正常系_endpointをユースケースへ渡して204を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushSubscriptionController(t)

			mockUsecase.EXPECT().Unsubscribe(gomock.Any(), uid, endpoint).Return(nil)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "DELETE", `{"endpoint":"`+endpoint+`"}`, uid, secretKey))

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_bodyが壊れていれば400", func(t *testing.T) {
			c, _, secretKey := setup4TestPushSubscriptionController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newPushSubscriptionRequest(t, "DELETE", `{`, uid, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
