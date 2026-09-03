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

func setup4TestChampionsleagueScheduleController(t *testing.T, r *gin.Engine) (
	*ChampionsleagueSchedule,
	*mock_usecase.MockChampionsleagueScheduleInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockChampionsleagueScheduleInterface(mockCtrl)

	c := NewChampionsleagueSchedule(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestChampionsleagueScheduleController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	_, mockUsecase := setup4TestChampionsleagueScheduleController(t, r)

	// JSONを往復させる時刻はUTCで固定する(CIはUTC、開発機はJSTのため)
	fromDate := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)

	schedule := &entity.ChampionsleagueSchedule{
		ID:       "cl2027_yokohama",
		Title:    "チャンピオンズリーグ2027 横浜",
		FromDate: fromDate,
		ToDate:   toDate,
	}

	t.Run("Get", func(t *testing.T) {
		t.Run("正常系_全大会を返す", func(t *testing.T) {
			mockUsecase.EXPECT().Find(gomock.Any()).Return([]*entity.ChampionsleagueSchedule{schedule}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/championsleague_schedules", nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var res []*dto.ChampionsleagueScheduleResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Len(t, res, 1)
			require.Equal(t, "cl2027_yokohama", res[0].ID)
			require.Equal(t, "チャンピオンズリーグ2027 横浜", res[0].Title)
		})

		t.Run("異常系_ユースケースがエラーを返した場合は500を返す", func(t *testing.T) {
			mockUsecase.EXPECT().Find(gomock.Any()).Return(nil, errors.New("unexpected error"))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/championsleague_schedules", nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetById", func(t *testing.T) {
		t.Run("正常系_指定IDの大会を返す", func(t *testing.T) {
			mockUsecase.EXPECT().FindById(gomock.Any(), "cl2027_yokohama").Return(schedule, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/championsleague_schedules/cl2027_yokohama", nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var res dto.ChampionsleagueScheduleResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, "cl2027_yokohama", res.ID)
		})

		t.Run("異常系_存在しないIDの場合は404を返す", func(t *testing.T) {
			mockUsecase.EXPECT().FindById(gomock.Any(), "unknown").Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/championsleague_schedules/unknown", nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})
}
