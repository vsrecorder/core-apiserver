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
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
)

func setup4TestStandardRegulationController(t *testing.T) (*StandardRegulation, *mock_usecase.MockStandardRegulationInterface) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockStandardRegulationInterface(mockCtrl)

	r := gin.Default()
	c := NewStandardRegulation(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestStandardRegulationController(t *testing.T) {
	regulation := &entity.StandardRegulation{ID: "regulation-g", Marks: "G,H,I"}

	t.Run("Get", func(t *testing.T) {
		t.Run("正常系_date未指定なら全レギュレーション一覧を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestStandardRegulationController(t)

			mockUsecase.EXPECT().Find(context.Background()).Return([]*entity.StandardRegulation{regulation}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", StandardRegulationsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestStandardRegulationController(t)

			mockUsecase.EXPECT().Find(context.Background()).Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", StandardRegulationsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetByDate", func(t *testing.T) {
		t.Run("正常系_date指定ならその日のレギュレーションを返す", func(t *testing.T) {
			c, mockUsecase := setup4TestStandardRegulationController(t)

			date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local)

			mockUsecase.EXPECT().FindByDate(context.Background(), date).Return(regulation, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", StandardRegulationsPath+"?date=2026-07-18", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_dateの形式が不正なら400を返す", func(t *testing.T) {
			c, _ := setup4TestStandardRegulationController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", StandardRegulationsPath+"?date=20260718", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_該当なしはErrRecordNotFoundから404を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestStandardRegulationController(t)

			mockUsecase.EXPECT().FindByDate(context.Background(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", StandardRegulationsPath+"?date=2000-01-01", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("GetById", func(t *testing.T) {
		t.Run("正常系_指定IDのレギュレーションを返す", func(t *testing.T) {
			c, mockUsecase := setup4TestStandardRegulationController(t)

			mockUsecase.EXPECT().FindById(context.Background(), "regulation-g").Return(regulation, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", StandardRegulationsPath+"/regulation-g", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
			c, mockUsecase := setup4TestStandardRegulationController(t)

			mockUsecase.EXPECT().FindById(context.Background(), "unknown").Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", StandardRegulationsPath+"/unknown", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})
}
