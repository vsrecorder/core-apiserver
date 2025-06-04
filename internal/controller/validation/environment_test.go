package validation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestEnvironmentValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"EnvironmentGetByDateMiddleware": test_EnvironmentGetByDateMiddleware,
		"EnvironmentGetByTermMiddleware": test_EnvironmentGetByTermMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_EnvironmentGetByDateMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByDateMiddleware()
		middleware(ginContext)

		expectedDate := time.Time{}

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedDate, helper.GetDate(ginContext))
	})

	t.Run("正常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		date := "2025-02-15"

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?date=%s", date), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByDateMiddleware()
		middleware(ginContext)

		expectedDate, _ := time.Parse(DateLayout, date)
		expectedDate = time.Date(expectedDate.Year(), expectedDate.Month(), expectedDate.Day(), 0, 0, 0, 0, time.UTC)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedDate, helper.GetDate(ginContext))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		date := "20250215"

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?date=%s", date), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByDateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		date := "0000-00-00"

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?date=%s", date), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByDateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_EnvironmentGetByTermMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByTermMiddleware()
		middleware(ginContext)

		expected := time.Time{}

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expected, helper.GetFromDate(ginContext))
		require.Equal(t, expected, helper.GetToDate(ginContext))
	})

	t.Run("正常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		fromDate := "2025-01-15"
		toDate := "2025-02-15"

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?from_date=%s&to_date=%s", fromDate, toDate), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByTermMiddleware()
		middleware(ginContext)

		expectedFromDate, _ := time.Parse(DateLayout, fromDate)
		expectedFromDate = time.Date(expectedFromDate.Year(), expectedFromDate.Month(), expectedFromDate.Day(), 0, 0, 0, 0, time.UTC)

		expectedToDate, _ := time.Parse(DateLayout, toDate)
		expectedToDate = time.Date(expectedToDate.Year(), expectedToDate.Month(), expectedToDate.Day(), 0, 0, 0, 0, time.UTC)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedFromDate, helper.GetFromDate(ginContext))
		require.Equal(t, expectedToDate, helper.GetToDate(ginContext))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		fromDate := "2025-01-15"
		//toDate := "2025-02-15"

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?from_date=%s", fromDate), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByTermMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		toDate := "2025-02-15"

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?to_date=%s", toDate), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByTermMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		fromDate := "2025-02-15"
		toDate := "2025-01-15"

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?from_date=%s&to_date=%s", fromDate, toDate), nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := EnvironmentGetByTermMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
