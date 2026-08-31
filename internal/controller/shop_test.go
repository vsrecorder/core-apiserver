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

func setup4TestShopController(t *testing.T) (*Shop, *mock_usecase.MockShopInterface) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockShopInterface(mockCtrl)

	r := gin.Default()
	c := NewShop(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestShopController(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		t.Run("正常系_キーワードで検索した店舗を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestShopController(t)

			shops := []*entity.Shop{
				{ID: 10317, Name: "古本市場新琴似店", PrefectureId: 1, PrefectureName: "北海道"},
			}

			mockUsecase.EXPECT().Find(gomock.Any(), "古本市場", 0).Return(shops, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ShopsPath+"?keyword=古本市場", nil)
			c.router.ServeHTTP(w, req)

			var res dto.ShopGetResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "古本市場", res.Keyword)
			require.Equal(t, 1, res.Count)
			require.Equal(t, uint(10317), res.Shops[0].ID)
			require.Equal(t, "北海道", res.Shops[0].PrefectureName)
		})

		t.Run("正常系_該当が無ければ空配列を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestShopController(t)

			mockUsecase.EXPECT().Find(gomock.Any(), "該当なし", 0).Return([]*entity.Shop{}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ShopsPath+"?keyword=該当なし", nil)
			c.router.ServeHTTP(w, req)

			var res dto.ShopGetResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, 0, res.Count)
			require.NotNil(t, res.Shops)
		})

		t.Run("異常系_キーワードが無ければ400を返す", func(t *testing.T) {
			c, _ := setup4TestShopController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ShopsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_キーワードが空白だけなら400を返す", func(t *testing.T) {
			c, _ := setup4TestShopController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ShopsPath+"?keyword=%20%20", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースが失敗したら500を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestShopController(t)

			mockUsecase.EXPECT().Find(gomock.Any(), "町田", 0).Return(nil, errors.New("error"))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ShopsPath+"?keyword=町田", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
