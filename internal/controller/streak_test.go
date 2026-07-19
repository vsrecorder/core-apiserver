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

func setup4TestStreakController(t *testing.T) (*Streak, *mock_usecase.MockStreakInterface) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockStreakInterface(mockCtrl)

	r := gin.Default()
	c := NewStreak(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestStreakController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_指定ユーザのストリークを返す", func(t *testing.T) {
		c, mockUsecase := setup4TestStreakController(t)

		streak := entity.NewUserStreak(uid, 3, 5, 1, 0, time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local), time.Now().Local())

		mockUsecase.EXPECT().GetByUserId(context.Background(), uid).Return(streak, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+StreakPath, nil)
		c.router.ServeHTTP(w, req)

		var res dto.UserStreakResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 3, res.CurrentWeeks)
		require.Equal(t, 5, res.LongestWeeks)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase := setup4TestStreakController(t)

		mockUsecase.EXPECT().GetByUserId(context.Background(), uid).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+StreakPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
