package controller

import (
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
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
)

func setup4TestChampionsleagueResultController(t *testing.T, r *gin.Engine) (
	*ChampionsleagueResult,
	*mock_usecase.MockChampionsleagueResultInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockChampionsleagueResultInterface(mockCtrl)

	c := NewChampionsleagueResult(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestChampionsleagueResultController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetEvents":                      test_ChampionsleagueResultController_GetEvents,
		"GetByChampionsleagueScheduleId": test_ChampionsleagueResultController_GetByChampionsleagueScheduleId,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_ChampionsleagueResultController_GetEvents(t *testing.T) {
	r := gin.Default()
	_, mockUsecase := setup4TestChampionsleagueResultController(t, r)

	eventDate, err := time.Parse(DateLayout, "2026-06-07")
	require.NoError(t, err)

	t.Run("正常系_結果が登録されているイベントを入賞者無しで返す", func(t *testing.T) {
		events := []*entity.ChampionsleagueResultEvent{
			{ChampionsleagueScheduleId: "pjcs2026", OfficialEventId: uint(1032135), LeagueType: uint(4), EventDate: eventDate},
			{ChampionsleagueScheduleId: "pjcs2026", OfficialEventId: uint(1032136), LeagueType: uint(3), EventDate: eventDate},
		}

		mockUsecase.EXPECT().FindEvents(gomock.Any()).Return(events, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/championsleague_results/events", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.ChampionsleagueResultGetEventsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, 2, res.Count)
		require.Len(t, res.Events, 2)
		require.Equal(t, "pjcs2026", res.Events[0].ChampionsleagueScheduleId)
		require.Equal(t, uint(1032135), res.Events[0].OfficialEventId)
		require.Equal(t, uint(4), res.Events[0].LeagueType)

		// 入賞者を含めないことがこのエンドポイントの目的なので、応答に results が現れないことを確かめる
		require.NotContains(t, w.Body.String(), "player_id")
		require.NotContains(t, w.Body.String(), "deck_code")
	})

	t.Run("異常系_ユースケースがエラーを返した場合は500を返す", func(t *testing.T) {
		mockUsecase.EXPECT().FindEvents(gomock.Any()).Return(nil, errors.New("unexpected error"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/championsleague_results/events", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_ChampionsleagueResultController_GetByChampionsleagueScheduleId(t *testing.T) {
	r := gin.Default()
	_, mockUsecase := setup4TestChampionsleagueResultController(t, r)

	eventDate, err := time.Parse(DateLayout, "2026-06-07")
	require.NoError(t, err)

	t.Run("正常系_大会IDでイベント単位にまとめた結果を返す", func(t *testing.T) {
		championsleagueResults := []*entity.ChampionsleagueResult{
			{
				ChampionsleagueScheduleId: "pjcs2026",
				OfficialEventId:           uint(1032135),
				LeagueType:                uint(4),
				EventDate:                 eventDate,
				EventResults: []*entity.ChampionsleagueEventResult{
					{PlayerId: "0123456789", PlayerName: "テスト太郎", Rank: uint(1), DeckCode: "xxxxxx-xxxxxx-xxxxxx"},
				},
			},
		}

		mockUsecase.EXPECT().FindByChampionsleagueScheduleId(
			gomock.Any(),
			uint(0),
			"pjcs2026",
		).Return(championsleagueResults, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/championsleague_results?championsleague_schedule_id=pjcs2026",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.ChampionsleagueResultGetByChampionsleagueScheduleIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, "pjcs2026", res.ChampionsleagueScheduleId)
		require.Equal(t, uint(0), res.LeagueType)
		require.Equal(t, 1, res.Count)
		require.Len(t, res.EventResults, 1)
		require.Equal(t, uint(1032135), res.EventResults[0].OfficialEventId)
		require.Equal(
			t,
			"https://players.pokemon-card.com/event/detail/1032135/result",
			res.EventResults[0].EventDetailResultURL,
		)
		require.Len(t, res.EventResults[0].Results, 1)
		require.Equal(t, "テスト太郎", res.EventResults[0].Results[0].PlayerName)
		require.Equal(t, uint(1), res.EventResults[0].Results[0].Rank)

		// championsleague_results に point 列は無いため、応答にも現れない
		require.NotContains(t, w.Body.String(), "point")
	})

	t.Run("正常系_league_typeで1区分に絞り込める", func(t *testing.T) {
		championsleagueResults := []*entity.ChampionsleagueResult{
			{
				ChampionsleagueScheduleId: "pjcs2026",
				OfficialEventId:           uint(1032135),
				LeagueType:                uint(4),
				EventDate:                 eventDate,
				EventResults: []*entity.ChampionsleagueEventResult{
					{PlayerId: "0123456789", PlayerName: "テスト太郎", Rank: uint(1), DeckCode: "xxxxxx-xxxxxx-xxxxxx"},
				},
			},
		}

		mockUsecase.EXPECT().FindByChampionsleagueScheduleId(
			gomock.Any(),
			uint(4),
			"pjcs2026",
		).Return(championsleagueResults, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/championsleague_results?championsleague_schedule_id=pjcs2026&league_type=4",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.ChampionsleagueResultGetByChampionsleagueScheduleIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, uint(4), res.LeagueType)
		require.Len(t, res.EventResults, 1)
		require.Equal(t, uint(4), res.EventResults[0].LeagueType)
	})

	t.Run("異常系_league_typeが範囲外の場合は400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/championsleague_results?championsleague_schedule_id=pjcs2026&league_type=5",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_championsleague_schedule_idが無い場合は400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/championsleague_results", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_結果が無い大会IDの場合は404を返す", func(t *testing.T) {
		mockUsecase.EXPECT().FindByChampionsleagueScheduleId(
			gomock.Any(),
			uint(0),
			"cl2027_yokohama",
		).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/championsleague_results?championsleague_schedule_id=cl2027_yokohama",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_ユースケースがエラーを返した場合は500を返す", func(t *testing.T) {
		mockUsecase.EXPECT().FindByChampionsleagueScheduleId(
			gomock.Any(),
			uint(0),
			"pjcs2026",
		).Return(nil, errors.New("unexpected error"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(
			http.MethodGet,
			"/championsleague_results?championsleague_schedule_id=pjcs2026",
			nil,
		)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
