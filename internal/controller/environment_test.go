package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func setupMock4TestEnvironmentController(t *testing.T) *mock_usecase.MockEnvironmentInterface {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockEnvironmentInterface(mockCtrl)

	return mockUsecase
}

func setup4TestEnvironmentController(t *testing.T, r *gin.Engine) (
	*Environment,
	*mock_usecase.MockEnvironmentInterface,
) {
	mockUsecase := setupMock4TestEnvironmentController(t)

	c := NewEnvironment(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestEnvironmentController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"Get":       test_EnvironmentController_Get,
		"GetById":   test_EnvironmentController_GetById,
		"GetByDate": test_EnvironmentController_GetByDate,
		"GetByTerm": test_EnvironmentController_GetByTerm,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_EnvironmentController_Get(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestEnvironmentController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		environment := entity.Environment{
			ID:       id,
			Title:    title,
			FromDate: fromDate,
			ToDate:   toDate,
		}

		environments := []*entity.Environment{
			&environment,
		}

		mockUsecase.EXPECT().Find(context.Background()).Return(environments, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", EnvironmentsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res []*dto.EnvironmentResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res[0].ID)
		require.Equal(t, title, res[0].Title)
		require.Equal(t, fromDate, res[0].FromDate)
		require.Equal(t, toDate, res[0].ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		mockUsecase.EXPECT().Find(context.Background()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", EnvironmentsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_EnvironmentController_GetById(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestEnvironmentController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		environment := entity.NewEnvironment(
			id,
			title,
			fromDate,
			toDate,
		)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(environment, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", EnvironmentsPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res *dto.EnvironmentResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, title, res.Title)
		require.Equal(t, fromDate, res.FromDate)
		require.Equal(t, toDate, res.ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id := "sv11"
		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", EnvironmentsPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id := "sv11"
		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", EnvironmentsPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_EnvironmentController_GetByDate(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestEnvironmentController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		date, _ := time.Parse(DateLayout, "2025-06-06")

		environment := entity.NewEnvironment(
			id,
			title,
			fromDate,
			toDate,
		)

		mockUsecase.EXPECT().FindByDate(context.Background(), date).Return(environment, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(EnvironmentsPath+"?date=%s", date.Format(DateLayout)), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res *dto.EnvironmentResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, title, res.Title)
		require.Equal(t, fromDate, res.FromDate)
		require.Equal(t, toDate, res.ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		date, _ := time.Parse(DateLayout, "2025-06-06")
		mockUsecase.EXPECT().FindByDate(context.Background(), date).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(EnvironmentsPath+"?date=%s", date.Format(DateLayout)), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		date, _ := time.Parse(DateLayout, "2025-06-06")
		mockUsecase.EXPECT().FindByDate(context.Background(), date).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(EnvironmentsPath+"?date=%s", date.Format(DateLayout)), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_EnvironmentController_GetByTerm(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestEnvironmentController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id := "sv11"
		title := "ブラックボルト/ホワイトフレア"
		fromDate, _ := time.Parse(DateLayout, "2025-06-06")
		toDate, _ := time.Parse(DateLayout, "2025-07-31")

		argFromDate, _ := time.Parse(DateLayout, "2025-06-06")
		argToDate, _ := time.Parse(DateLayout, "2025-06-07")

		environment := entity.Environment{
			ID:       id,
			Title:    title,
			FromDate: fromDate,
			ToDate:   toDate,
		}

		environments := []*entity.Environment{
			&environment,
		}

		mockUsecase.EXPECT().FindByTerm(context.Background(), argFromDate, argToDate).Return(environments, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(EnvironmentsPath+"?from_date=%s&to_date=%s", argFromDate.Format(DateLayout), argToDate.Format(DateLayout)), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res []*dto.EnvironmentResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res[0].ID)
		require.Equal(t, title, res[0].Title)
		require.Equal(t, fromDate, res[0].FromDate)
		require.Equal(t, toDate, res[0].ToDate)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		argFromDate, _ := time.Parse(DateLayout, "2025-06-06")
		argToDate, _ := time.Parse(DateLayout, "2025-06-07")
		mockUsecase.EXPECT().FindByTerm(context.Background(), argFromDate, argToDate).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(EnvironmentsPath+"?from_date=%s&to_date=%s", argFromDate.Format(DateLayout), argToDate.Format(DateLayout)), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		argFromDate, _ := time.Parse(DateLayout, "2025-06-06")
		argToDate, _ := time.Parse(DateLayout, "2025-06-07")
		mockUsecase.EXPECT().FindByTerm(context.Background(), argFromDate, argToDate).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(EnvironmentsPath+"?from_date=%s&to_date=%s", argFromDate.Format(DateLayout), argToDate.Format(DateLayout)), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
