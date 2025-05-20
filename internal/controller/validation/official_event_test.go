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

func TestOfficialEventValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"OfficialEventGetMiddleware":     test_OfficialEventGetMiddleware,
		"OfficialEventGetByIdMiddleware": test_OfficialEventGetByIdMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_OfficialEventGetMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		startDate := "2025-02-15"
		endDate := "2025-02-15"

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?start_date=%s&end_date=%s", startDate, endDate), nil)
		require.NoError(t, err)

		expectedTypeId := uint(0)
		expectedLeagueType := uint(0)
		expectedStartDate, _ := time.Parse(DateLayout, startDate)
		expectedStartDate = time.Date(expectedStartDate.Year(), expectedStartDate.Month(), expectedStartDate.Day(), 0, 0, 0, 0, time.UTC)
		expectedEndDate, _ := time.Parse(DateLayout, endDate)
		expectedEndDate = time.Date(expectedEndDate.Year(), expectedEndDate.Month(), expectedEndDate.Day(), 0, 0, 0, 0, time.UTC)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedTypeId, helper.GetTypeId(ginContext))
		require.Equal(t, expectedLeagueType, helper.GetLeagueType(ginContext))
		require.Equal(t, expectedStartDate, helper.GetStartDate(ginContext))
		require.Equal(t, expectedEndDate, helper.GetEndDate(ginContext))
	})

	t.Run("正常系_#02", func(t *testing.T) {
		startDate := "2025-02-15"
		endDate := "2025-02-16"

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?start_date=%s&end_date=%s", startDate, endDate), nil)
		require.NoError(t, err)

		expectedTypeId := uint(0)
		expectedLeagueType := uint(0)
		expectedStartDate, _ := time.Parse(DateLayout, startDate)
		expectedStartDate = time.Date(expectedStartDate.Year(), expectedStartDate.Month(), expectedStartDate.Day(), 0, 0, 0, 0, time.UTC)
		expectedEndDate, _ := time.Parse(DateLayout, endDate)
		expectedEndDate = time.Date(expectedEndDate.Year(), expectedEndDate.Month(), expectedEndDate.Day(), 0, 0, 0, 0, time.UTC)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedTypeId, helper.GetTypeId(ginContext))
		require.Equal(t, expectedLeagueType, helper.GetLeagueType(ginContext))
		require.Equal(t, expectedStartDate, helper.GetStartDate(ginContext))
		require.Equal(t, expectedEndDate, helper.GetEndDate(ginContext))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		startDate := "2025-02-16"
		endDate := "2025-02-15"

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?start_date=%s&end_date=%s", startDate, endDate), nil)
		require.NoError(t, err)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_OfficialEventGetByIdMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		id := uint(10000)

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: "10000",
			},
		)

		middleware := OfficialEventGetByIdMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, helper.GetOfficialEventId(ginContext))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: "a",
			},
		)

		middleware := OfficialEventGetByIdMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: "0",
			},
		)

		middleware := OfficialEventGetByIdMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: "-1",
			},
		)

		middleware := OfficialEventGetByIdMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
