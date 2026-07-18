package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

// stubCalendarUsecase はカレンダーユースケースのスタブ。
// mock_usecaseにCalendar用のモックが存在しないため手書きする。
type stubCalendarUsecase struct {
	calendar *entity.Calendar
	err      error
}

func (s stubCalendarUsecase) GetCalendar(ctx context.Context, userId string) (*entity.Calendar, error) {
	return s.calendar, s.err
}

func setup4TestCalendarController(t *testing.T, u stubCalendarUsecase) (*Calendar, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	r := gin.Default()
	c := NewCalendar(r, u)
	c.RegisterRoute("")

	return c, secretKey
}

func TestCalendarController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_本人ならカレンダーを返す", func(t *testing.T) {
		c, secretKey := setup4TestCalendarController(t, stubCalendarUsecase{calendar: &entity.Calendar{}})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+CalendarPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _ := setup4TestCalendarController(t, stubCalendarUsecase{calendar: &entity.Calendar{}})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+CalendarPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人のカレンダーは403を返す", func(t *testing.T) {
		c, secretKey := setup4TestCalendarController(t, stubCalendarUsecase{calendar: &entity.Calendar{}})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+CalendarPath, nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, secretKey := setup4TestCalendarController(t, stubCalendarUsecase{err: errors.New("")})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+CalendarPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
