package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestDeckUsageStatController(t *testing.T) (*DeckUsageStat, *mock_usecase.MockDeckUsageStatInterface, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockDeckUsageStatInterface(mockCtrl)

	r := gin.Default()
	c := NewDeckUsageStat(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestDeckUsageStatController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_本人なら集計条件を渡してデッキ使用統計を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckUsageStat(context.Background(), uid, "2026-07", "", "", "", true).
			Return(&entity.DeckUsageStat{}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath+"?year_month=2026-07&all_time=true", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人の統計は403を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_該当なしはErrRecordNotFoundから404を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckUsageStat(context.Background(), uid, "", "", "", "", false).
			Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckUsageStat(context.Background(), uid, "", "", "", "", false).
			Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
