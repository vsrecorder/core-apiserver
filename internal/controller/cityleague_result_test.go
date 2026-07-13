package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
)

func setup4TestCityleagueResultController(t *testing.T, r *gin.Engine) (
	*CityleagueResult,
	*mock_usecase.MockCityleagueResultInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockCityleagueResultInterface(mockCtrl)

	c := NewCityleagueResult(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestCityleagueResultController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetEvents": test_CityleagueResultController_GetEvents,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_CityleagueResultController_GetEvents(t *testing.T) {
	r := gin.Default()
	_, mockUsecase := setup4TestCityleagueResultController(t, r)

	eventDate, err := time.Parse(DateLayout, "2026-04-30")
	require.NoError(t, err)

	t.Run("正常系_#01_クエリパラメータ無しの場合は全リーグ・全期間が対象になる", func(t *testing.T) {
		cityleagueResultEvents := []*entity.CityleagueResultEvent{
			{OfficialEventId: uint(952749), LeagueType: uint(1), EventDate: eventDate},
			{OfficialEventId: uint(952750), LeagueType: uint(2), EventDate: eventDate},
		}

		mockUsecase.EXPECT().FindEvents(
			context.Background(),
			uint(0),
			time.Time{},
			time.Time{},
		).Return(cityleagueResultEvents, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/cityleague_results/events", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.CityleagueResultGetEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, 2, res.Count)
		require.Len(t, res.Events, 2)
		require.Equal(t, uint(952749), res.Events[0].OfficialEventId)
		require.Equal(t, uint(1), res.Events[0].LeagueType)
		require.Equal(t, uint(952750), res.Events[1].OfficialEventId)

		// 入賞者を含めないことがこのエンドポイントの目的なので、応答に results が現れないことを確かめる
		require.NotContains(t, w.Body.String(), "player_id")
		require.NotContains(t, w.Body.String(), "deck_code")
	})

	t.Run("正常系_#02_league_typeと期間で絞り込める", func(t *testing.T) {
		// helper のクエリパースは日付をローカルタイム（JST）として解釈するため、期待値もそれに合わせる
		fromDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 4, 30, 0, 0, 0, 0, time.Local)

		cityleagueResultEvents := []*entity.CityleagueResultEvent{
			{OfficialEventId: uint(952749), LeagueType: uint(1), EventDate: eventDate},
		}

		mockUsecase.EXPECT().FindEvents(
			context.Background(),
			uint(1),
			fromDate,
			toDate,
		).Return(cityleagueResultEvents, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/cityleague_results/events?league_type=1&from_date=2026-04-01&to_date=2026-04-30",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.CityleagueResultGetEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, 1, res.Count)
		require.Len(t, res.Events, 1)
		require.Equal(t, uint(952749), res.Events[0].OfficialEventId)
	})

	t.Run("異常系_#01_from_dateのみ指定した場合は400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/cityleague_results/events?from_date=2026-04-01",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02_from_dateがto_dateより後の場合は400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/cityleague_results/events?from_date=2026-04-30&to_date=2026-04-01",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#03_ユースケースがエラーを返した場合は500を返す", func(t *testing.T) {
		mockUsecase.EXPECT().FindEvents(
			context.Background(),
			uint(0),
			time.Time{},
			time.Time{},
		).Return(nil, errors.New("unexpected error"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/cityleague_results/events", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
