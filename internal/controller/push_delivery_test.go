package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestPushDeliveryController(t *testing.T) (
	*PushDelivery,
	*mock_usecase.MockPushDeliveryInterface,
	string,
) {
	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockPushDeliveryInterface(mockCtrl)

	r := gin.Default()
	c := NewPushDelivery(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestPushDeliveryController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("MarkDelivered", func(t *testing.T) {
		t.Run("正常系_本人のidを記録して204を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushDeliveryController(t)

			mockUsecase.EXPECT().MarkDelivered(gomock.Any(), uid, id).Return(nil)

			req, _ := http.NewRequest("POST", UsersPath+PushDeliveriesPath+"/"+id+"/delivered", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_未認証は401", func(t *testing.T) {
			c, _, _ := setup4TestPushDeliveryController(t)

			req, _ := http.NewRequest("POST", UsersPath+PushDeliveriesPath+"/"+id+"/delivered", nil)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("MarkClicked", func(t *testing.T) {
		t.Run("正常系_本人のidを記録して204を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushDeliveryController(t)

			mockUsecase.EXPECT().MarkClicked(gomock.Any(), uid, id).Return(nil)

			req, _ := http.NewRequest("POST", UsersPath+PushDeliveriesPath+"/"+id+"/clicked", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_他人の配達ログはErrRecordNotFoundから404を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushDeliveryController(t)

			mockUsecase.EXPECT().MarkClicked(gomock.Any(), uid, id).Return(apperror.ErrRecordNotFound)

			req, _ := http.NewRequest("POST", UsersPath+PushDeliveriesPath+"/"+id+"/clicked", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestPushDeliveryController(t)

			mockUsecase.EXPECT().MarkClicked(gomock.Any(), uid, id).Return(errors.New("db down"))

			req, _ := http.NewRequest("POST", UsersPath+PushDeliveriesPath+"/"+id+"/clicked", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
