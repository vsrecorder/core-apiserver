package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestOldestRecordController(t *testing.T) (*OldestRecord, *mock_usecase.MockOldestRecordInterface, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockOldestRecordInterface(mockCtrl)

	r := gin.Default()
	c := NewOldestRecord(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestOldestRecordController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_本人なら最古の記録日を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOldestRecordController(t)

		eventDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

		mockUsecase.EXPECT().GetOldestRecord(context.Background(), uid, "").Return(entity.NewOldestRecord(&eventDate), nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OldestRecordPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		var res dto.OldestRecordResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, uid, res.UserId)
		require.NotNil(t, res.EventDate)
	})

	t.Run("正常系_deck_id指定はそのままユースケースへ渡す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOldestRecordController(t)

		deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		mockUsecase.EXPECT().GetOldestRecord(context.Background(), uid, deckId).Return(entity.NewOldestRecord(nil), nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OldestRecordPath+"?deck_id="+deckId, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		var res dto.OldestRecordResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, deckId, res.DeckId)
		require.Nil(t, res.EventDate)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestOldestRecordController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OldestRecordPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人の記録は403を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOldestRecordController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OldestRecordPath, nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOldestRecordController(t)

		mockUsecase.EXPECT().GetOldestRecord(context.Background(), uid, "").Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+OldestRecordPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
