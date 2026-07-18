package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestNotificationController(t *testing.T, r *gin.Engine) (
	*Notification,
	*mock_usecase.MockNotificationInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockNotificationInterface(mockCtrl)

	c := NewNotification(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestNotificationController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetByUID":      test_NotificationController_GetByUID,
		"CountUnread":   test_NotificationController_CountUnread,
		"MarkAsRead":    test_NotificationController_MarkAsRead,
		"MarkAllAsRead": test_NotificationController_MarkAllAsRead,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_NotificationController_GetByUID(t *testing.T) {
	r := gin.Default()

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	c, mockUsecase := setup4TestNotificationController(t, r)

	t.Run("正常系", func(t *testing.T) {
		notifications := []*entity.Notification{
			entity.NewNotification("n-1", time.Now(), uid, "badge", "バッジを獲得しました", "「初記録」バッジを獲得しました！", "/users"),
		}

		mockUsecase.EXPECT().ListByUserId(context.Background(), uid, gomock.Any()).Return(notifications, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", NotificationsPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.NotificationsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		require.Len(t, res.Notifications, 1)
		require.Equal(t, "n-1", res.Notifications[0].ID)
	})

	t.Run("異常系_未認証は401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", NotificationsPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func test_NotificationController_CountUnread(t *testing.T) {
	r := gin.Default()

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	c, mockUsecase := setup4TestNotificationController(t, r)

	t.Run("正常系", func(t *testing.T) {
		mockUsecase.EXPECT().CountUnreadByUserId(context.Background(), uid).Return(2, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", NotificationsPath+"/unread_count", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.UnreadCountResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		require.Equal(t, 2, res.UnreadCount)
	})
}

func test_NotificationController_MarkAsRead(t *testing.T) {
	r := gin.Default()

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	c, mockUsecase := setup4TestNotificationController(t, r)

	t.Run("正常系", func(t *testing.T) {
		mockUsecase.EXPECT().MarkAsRead(context.Background(), uid, "n-1").Return(nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", NotificationsPath+"/n-1/read", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_他人の通知や存在しないIDは404", func(t *testing.T) {
		mockUsecase.EXPECT().MarkAsRead(context.Background(), uid, "n-2").Return(apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", NotificationsPath+"/n-2/read", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_その他のエラーは500", func(t *testing.T) {
		mockUsecase.EXPECT().MarkAsRead(context.Background(), uid, "n-3").Return(errors.New("db error"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", NotificationsPath+"/n-3/read", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_NotificationController_MarkAllAsRead(t *testing.T) {
	r := gin.Default()

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	c, mockUsecase := setup4TestNotificationController(t, r)

	t.Run("正常系", func(t *testing.T) {
		mockUsecase.EXPECT().MarkAllAsRead(context.Background(), uid).Return(nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", NotificationsPath+"/read_all", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})
}
