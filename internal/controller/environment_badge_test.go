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
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func setup4TestEnvironmentBadgeController(t *testing.T) (*EnvironmentBadge, *mock_usecase.MockEnvironmentBadgeInterface) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockEnvironmentBadgeInterface(mockCtrl)

	r := gin.Default()
	c := NewEnvironmentBadge(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestEnvironmentBadgeController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_指定ユーザの環境バッジ一覧を返す", func(t *testing.T) {
		c, mockUsecase := setup4TestEnvironmentBadgeController(t)

		views := []*usecase.EnvironmentBadgeView{
			{
				Environment: entity.NewEnvironment("sv11", "ブラックボルト/ホワイトフレア", time.Now(), time.Now()),
				Achieved:    true,
				AchievedAt:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local),
			},
		}

		mockUsecase.EXPECT().GetByUserId(context.Background(), uid).Return(views, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+EnvironmentBadgesPath, nil)
		c.router.ServeHTTP(w, req)

		var res dto.UserEnvironmentBadgesResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, uid, res.UserId)
		require.Len(t, res.Badges, 1)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase := setup4TestEnvironmentBadgeController(t)

		mockUsecase.EXPECT().GetByUserId(context.Background(), uid).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+EnvironmentBadgesPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
