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

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestOpponentDeckUsageStatController(t *testing.T) (*OpponentDeckUsageStat, *mock_usecase.MockOpponentDeckUsageStatInterface, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockOpponentDeckUsageStatInterface(mockCtrl)

	r := gin.Default()
	c := NewOpponentDeckUsageStat(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestOpponentDeckUsageStatController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_本人なら集計条件を渡して相手デッキ統計を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckUsageStatController(t)

		deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		mockUsecase.EXPECT().GetOpponentDeckUsageStat(context.Background(), uid, "2026-07", "", "", "", deckId).
			Return(&entity.OpponentDeckUsageStat{}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OpponentDeckUsageStatsPath+"?year_month=2026-07&deck_id="+deckId, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestOpponentDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OpponentDeckUsageStatsPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人の統計は403を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OpponentDeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckUsageStatController(t)

		mockUsecase.EXPECT().GetOpponentDeckUsageStat(context.Background(), uid, "", "", "", "", "").
			Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OpponentDeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
