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
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
)

func setup4TestWeeklyDeckUsageStatController(t *testing.T) (*WeeklyDeckUsageStat, *mock_usecase.MockWeeklyDeckUsageStatInterface) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockWeeklyDeckUsageStatInterface(mockCtrl)

	r := gin.Default()
	c := NewWeeklyDeckUsageStat(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestWeeklyDeckUsageStatController_GetWeeklyUsage(t *testing.T) {
	t.Run("正常系_指定週の統計を返す", func(t *testing.T) {
		c, mockUsecase := setup4TestWeeklyDeckUsageStatController(t)

		weekStart := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
		stat := entity.NewWeeklyDeckUsageStat(weekStart, 10, 3, []*entity.DeckUsageVariant{})

		mockUsecase.EXPECT().GetWeeklyDeckUsageStat(context.Background(), "2026-07-13").Return(stat, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", DeckMetaPath+WeeklyDeckUsagePath+"?week=2026-07-13", nil)
		c.router.ServeHTTP(w, req)

		var res dto.WeeklyDeckUsageStatResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 10, res.TotalVotes)
		require.Equal(t, 3, res.ContributorCount)
	})

	t.Run("異常系_weekの形式が不正なら400を返す", func(t *testing.T) {
		c, _ := setup4TestWeeklyDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", DeckMetaPath+WeeklyDeckUsagePath+"?week=2026/07/13", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase := setup4TestWeeklyDeckUsageStatController(t)

		mockUsecase.EXPECT().GetWeeklyDeckUsageStat(context.Background(), "").Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", DeckMetaPath+WeeklyDeckUsagePath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
