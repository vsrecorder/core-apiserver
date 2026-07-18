package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
)

func setup4TestUserStatController(t *testing.T) (
	*UserStat,
	*mock_usecase.MockUserStatInterface,
	*mock_usecase.MockUserStatHistoryInterface,
	*mock_usecase.MockUserStatRecentInterface,
) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockUserStatInterface(mockCtrl)
	mockHistoryUsecase := mock_usecase.NewMockUserStatHistoryInterface(mockCtrl)
	mockRecentUsecase := mock_usecase.NewMockUserStatRecentInterface(mockCtrl)

	r := gin.Default()
	c := NewUserStat(r, mockUsecase, mockHistoryUsecase, mockRecentUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, mockHistoryUsecase, mockRecentUsecase
}

func TestUserStatController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("GetByUserId", func(t *testing.T) {
		t.Run("正常系_集計条件をユースケースへ渡して統計を返す", func(t *testing.T) {
			c, mockUsecase, _, _ := setup4TestUserStatController(t)

			stat := entity.NewUserStat(uid, 5, 2, 1, 1, 10, 6, 4, 0.6)

			mockUsecase.EXPECT().GetUserStat(context.Background(), uid, "2026-07", "sv11", "", "").Return(stat, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"?year_month=2026-07&environment_id=sv11", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_year_monthの形式が不正なら400を返す", func(t *testing.T) {
			c, _, _, _ := setup4TestUserStatController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"?year_month=abc", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_該当なしはErrRecordNotFoundから404を返す", func(t *testing.T) {
			c, mockUsecase, _, _ := setup4TestUserStatController(t)

			mockUsecase.EXPECT().GetUserStat(context.Background(), uid, "", "", "", "").Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, _, _ := setup4TestUserStatController(t)

			mockUsecase.EXPECT().GetUserStat(context.Background(), uid, "", "", "", "").Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetHistoryByUserId", func(t *testing.T) {
		t.Run("正常系_期間指定で月次の履歴を返す", func(t *testing.T) {
			c, _, mockHistoryUsecase, _ := setup4TestUserStatController(t)

			history := []*entity.UserStatMonthly{entity.NewUserStatMonthly("2026-06", 4, 3, 1, 0.75)}

			mockHistoryUsecase.EXPECT().GetUserStatHistory(context.Background(), uid, "6months", "", "").Return(history, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history?period=6months", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_未定義のperiodなら400を返す", func(t *testing.T) {
			c, _, _, _ := setup4TestUserStatController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history?period=1year", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, _, mockHistoryUsecase, _ := setup4TestUserStatController(t)

			mockHistoryUsecase.EXPECT().GetUserStatHistory(context.Background(), uid, "3months", "", "").Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetRecentByUserId", func(t *testing.T) {
		t.Run("正常系_指定件数の直近対戦の統計を返す", func(t *testing.T) {
			c, _, _, mockRecentUsecase := setup4TestUserStatController(t)

			mockRecentUsecase.EXPECT().GetRecentMatches(context.Background(), uid, 50, "").Return(&entity.RecentMatchStat{}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/recent?count=50", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_未定義のcountなら400を返す", func(t *testing.T) {
			c, _, _, _ := setup4TestUserStatController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/recent?count=25", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, _, _, mockRecentUsecase := setup4TestUserStatController(t)

			mockRecentUsecase.EXPECT().GetRecentMatches(context.Background(), uid, 20, "").Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/recent", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
