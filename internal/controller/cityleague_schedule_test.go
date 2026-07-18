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

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// mock_usecaseにCityleagueSchedule用のモックが存在しないため、同じメソッドを持つ
// 手書きスタブをusecaseインターフェースとして使う。
type stubCityleagueScheduleUsecase struct {
	schedules []*entity.CityleagueSchedule
	schedule  *entity.CityleagueSchedule
	err       error
}

func (s stubCityleagueScheduleUsecase) Find(ctx context.Context) ([]*entity.CityleagueSchedule, error) {
	return s.schedules, s.err
}

func (s stubCityleagueScheduleUsecase) FindById(ctx context.Context, id string) (*entity.CityleagueSchedule, error) {
	return s.schedule, s.err
}

func (s stubCityleagueScheduleUsecase) FindByDate(ctx context.Context, date time.Time) (*entity.CityleagueSchedule, error) {
	return s.schedule, s.err
}

func setup4TestCityleagueScheduleController(t *testing.T, u stubCityleagueScheduleUsecase) *CityleagueSchedule {
	t.Helper()

	gin.SetMode(gin.TestMode)

	r := gin.Default()
	c := NewCityleagueSchedule(r, u)
	c.RegisterRoute("")

	return c
}

func TestCityleagueScheduleController(t *testing.T) {
	schedule := &entity.CityleagueSchedule{ID: "2026_s1", Title: "シティリーグ2026 シーズン1"}

	t.Run("正常系_date未指定なら全日程一覧を返す", func(t *testing.T) {
		c := setup4TestCityleagueScheduleController(t, stubCityleagueScheduleUsecase{schedules: []*entity.CityleagueSchedule{schedule}})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", CityleagueSchedulesPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("正常系_date指定ならその日が属する日程を返す", func(t *testing.T) {
		c := setup4TestCityleagueScheduleController(t, stubCityleagueScheduleUsecase{schedule: schedule})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", CityleagueSchedulesPath+"?date=2026-07-18", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("正常系_指定IDの日程を返す", func(t *testing.T) {
		c := setup4TestCityleagueScheduleController(t, stubCityleagueScheduleUsecase{schedule: schedule})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", CityleagueSchedulesPath+"/2026_s1", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_dateの形式が不正なら400を返す", func(t *testing.T) {
		c := setup4TestCityleagueScheduleController(t, stubCityleagueScheduleUsecase{})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", CityleagueSchedulesPath+"?date=20260718", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
		c := setup4TestCityleagueScheduleController(t, stubCityleagueScheduleUsecase{err: apperror.ErrRecordNotFound})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", CityleagueSchedulesPath+"/unknown", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c := setup4TestCityleagueScheduleController(t, stubCityleagueScheduleUsecase{err: errors.New("")})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", CityleagueSchedulesPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
