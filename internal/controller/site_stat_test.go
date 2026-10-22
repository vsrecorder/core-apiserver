package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
)

func setup4TestSiteStatController(t *testing.T) (*SiteStat, *mock_usecase.MockSiteStatInterface) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockSiteStatInterface(mockCtrl)

	r := gin.Default()
	c := NewSiteStat(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestSiteStatController(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		t.Run("正常系_認証なしで3つの件数を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestSiteStatController(t)

			mockUsecase.EXPECT().Find(gomock.Any()).Return(entity.NewSiteStat(812, 15234, 4021), nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", SiteStatsPath, nil)
			c.router.ServeHTTP(w, req)

			var res dto.SiteStatResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, int64(812), res.UserCount)
			require.Equal(t, int64(15234), res.MatchCount)
			require.Equal(t, int64(4021), res.DeckCodeCount)
		})

		t.Run("異常系_集計に失敗したら500を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestSiteStatController(t)

			mockUsecase.EXPECT().Find(gomock.Any()).Return(nil, errors.New("connection refused"))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", SiteStatsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
