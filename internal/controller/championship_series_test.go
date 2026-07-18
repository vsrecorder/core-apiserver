package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// mock_usecaseにChampionshipSeries用のモックが存在しないため、同じメソッドを持つ
// mock_repositoryのモックをusecaseインターフェースとして使う。
func setup4TestChampionshipSeriesController(t *testing.T) (*ChampionshipSeries, *mock_repository.MockChampionshipSeriesInterface) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	r := gin.Default()
	c := NewChampionshipSeries(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestChampionshipSeriesController(t *testing.T) {
	series := &entity.ChampionshipSeries{ID: "series_2026", Title: "チャンピオンシップシリーズ2026"}

	t.Run("Get", func(t *testing.T) {
		t.Run("正常系_date未指定なら全シリーズ一覧を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestChampionshipSeriesController(t)

			mockUsecase.EXPECT().Find(context.Background()).Return([]*entity.ChampionshipSeries{series}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ChampionshipSeriesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestChampionshipSeriesController(t)

			mockUsecase.EXPECT().Find(context.Background()).Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ChampionshipSeriesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetByDate", func(t *testing.T) {
		t.Run("正常系_date指定ならその日が属するシリーズを返す", func(t *testing.T) {
			c, mockUsecase := setup4TestChampionshipSeriesController(t)

			date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local)

			mockUsecase.EXPECT().FindByDate(context.Background(), date).Return(series, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ChampionshipSeriesPath+"?date=2026-07-18", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_dateの形式が不正なら400を返す", func(t *testing.T) {
			c, _ := setup4TestChampionshipSeriesController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ChampionshipSeriesPath+"?date=20260718", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("GetById", func(t *testing.T) {
		t.Run("正常系_指定IDのシリーズを返す", func(t *testing.T) {
			c, mockUsecase := setup4TestChampionshipSeriesController(t)

			mockUsecase.EXPECT().FindById(context.Background(), "series_2026").Return(series, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ChampionshipSeriesPath+"/series_2026", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestChampionshipSeriesController(t)

			mockUsecase.EXPECT().FindById(context.Background(), "series_9999").Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", ChampionshipSeriesPath+"/series_9999", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})
}
