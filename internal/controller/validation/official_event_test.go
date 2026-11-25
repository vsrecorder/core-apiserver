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
	t.Run("正常系_開始日と終了日が同日の期間指定を受け付ける", func(t *testing.T) {
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
		expectedStartDate = time.Date(expectedStartDate.Year(), expectedStartDate.Month(), expectedStartDate.Day(), 0, 0, 0, 0, time.Local)
		expectedEndDate, _ := time.Parse(DateLayout, endDate)
		expectedEndDate = time.Date(expectedEndDate.Year(), expectedEndDate.Month(), expectedEndDate.Day(), 0, 0, 0, 0, time.Local)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedTypeId, helper.GetTypeId(ginContext))
		require.Equal(t, expectedLeagueType, helper.GetLeagueType(ginContext))
		require.Equal(t, time.Time{}, helper.GetDate(ginContext))
		require.Equal(t, expectedStartDate, helper.GetStartDate(ginContext))
		require.Equal(t, expectedEndDate, helper.GetEndDate(ginContext))
	})

	t.Run("正常系_開始日より後の終了日の期間指定を受け付ける", func(t *testing.T) {
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
		expectedStartDate = time.Date(expectedStartDate.Year(), expectedStartDate.Month(), expectedStartDate.Day(), 0, 0, 0, 0, time.Local)
		expectedEndDate, _ := time.Parse(DateLayout, endDate)
		expectedEndDate = time.Date(expectedEndDate.Year(), expectedEndDate.Month(), expectedEndDate.Day(), 0, 0, 0, 0, time.Local)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedTypeId, helper.GetTypeId(ginContext))
		require.Equal(t, expectedLeagueType, helper.GetLeagueType(ginContext))
		require.Equal(t, time.Time{}, helper.GetDate(ginContext))
		require.Equal(t, expectedStartDate, helper.GetStartDate(ginContext))
		require.Equal(t, expectedEndDate, helper.GetEndDate(ginContext))
	})

	t.Run("正常系_date単独指定を受け付ける", func(t *testing.T) {
		date := "2025-02-15"

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", fmt.Sprintf("/?date=%s", date), nil)
		require.NoError(t, err)

		expectedTypeId := uint(0)
		expectedLeagueType := uint(0)
		expectedDate, _ := time.Parse(DateLayout, date)
		expectedDate = time.Date(expectedDate.Year(), expectedDate.Month(), expectedDate.Day(), 0, 0, 0, 0, time.Local)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, expectedTypeId, helper.GetTypeId(ginContext))
		require.Equal(t, expectedLeagueType, helper.GetLeagueType(ginContext))
		require.Equal(t, expectedDate, helper.GetDate(ginContext))
	})

	// 期間幅の上限。開始日と終了日の差が上限ちょうどまでは受け付ける
	t.Run("正常系_上限ちょうどの期間指定を受け付ける", func(t *testing.T) {
		startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
		endDate := startDate.AddDate(0, 0, officialEventsRangeMaxDays)

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		req, err := http.NewRequest("GET", fmt.Sprintf(
			"/?type_id=2&start_date=%s&end_date=%s", startDate.Format(DateLayout), endDate.Format(DateLayout),
		), nil)
		require.NoError(t, err)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, startDate, helper.GetStartDate(ginContext))
		require.Equal(t, endDate, helper.GetEndDate(ginContext))
	})

	// 上限を1日でも超えたら、DB を引く前に断る
	t.Run("異常系_上限を超える期間指定なら400を返す", func(t *testing.T) {
		startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
		endDate := startDate.AddDate(0, 0, officialEventsRangeMaxDays+1)

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		req, err := http.NewRequest("GET", fmt.Sprintf(
			"/?type_id=2&start_date=%s&end_date=%s", startDate.Format(DateLayout), endDate.Format(DateLayout),
		), nil)
		require.NoError(t, err)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 全期間を1回で引こうとする指定(この上限を入れた理由そのもの)
	t.Run("異常系_全期間の指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		req, err := http.NewRequest("GET", "/?start_date=2018-01-01&end_date=2030-12-31", nil)
		require.NoError(t, err)

		ginContext.Request = req
		middleware := OfficialEventGetMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_start_dateがend_dateより後なら400を返す", func(t *testing.T) {
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
	t.Run("正常系_数値のIDをパスパラメータから設定する", func(t *testing.T) {
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

	t.Run("異常系_IDが数値でなければ400を返す", func(t *testing.T) {
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

	t.Run("異常系_IDが0なら400を返す", func(t *testing.T) {
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

	t.Run("異常系_IDが負数なら400を返す", func(t *testing.T) {
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
