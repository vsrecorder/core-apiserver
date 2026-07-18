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
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func setup4TestDesignationController(t *testing.T) (
	*Designation,
	*mock_usecase.MockDesignationInterface,
	*mock_repository.MockChampionshipSeriesInterface,
) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockDesignationInterface(mockCtrl)
	mockSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	r := gin.Default()
	c := NewDesignation(r, mockUsecase, mockSeriesRepo)
	c.RegisterRoute("")

	return c, mockUsecase, mockSeriesRepo
}

func newTestDesignation(id string, tier int) *entity.Designation {
	now := time.Now().Local()
	return entity.NewDesignation(id, tier, "rookie", "🔰", "ルーキー", "記録を1件作成", "record_count", 1, now, now)
}

func TestDesignationController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("GetAllDefinitions", func(t *testing.T) {
		t.Run("正常系_称号定義一覧を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestDesignationController(t)

			mockUsecase.EXPECT().GetAllDefinitions(context.Background()).Return(
				[]*entity.Designation{newTestDesignation("designation-rookie", 1)}, nil,
			)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DesignationsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestDesignationController(t)

			mockUsecase.EXPECT().GetAllDefinitions(context.Background()).Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DesignationsPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetByUserId", func(t *testing.T) {
		t.Run("正常系_season指定でユーザの称号を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestDesignationController(t)

			view := &usecase.UserDesignationView{Current: newTestDesignation("designation-rookie", 1)}

			mockUsecase.EXPECT().GetByUserId(context.Background(), uid, "2026").Return(view, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DesignationPath+"?season=2026", nil)
			c.router.ServeHTTP(w, req)

			var res dto.UserDesignationResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uid, res.UserId)
			require.Equal(t, "2026", res.Season)
		})

		t.Run("正常系_season未指定なら現在のシーズンで判定する", func(t *testing.T) {
			c, mockUsecase, mockSeriesRepo := setup4TestDesignationController(t)

			mockSeriesRepo.EXPECT().FindByDate(context.Background(), gomock.Any()).Return(
				&entity.ChampionshipSeries{ID: "series_2026"}, nil,
			)
			mockUsecase.EXPECT().GetByUserId(context.Background(), uid, "2026").Return(&usecase.UserDesignationView{}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DesignationPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_seasonの形式が不正なら400を返す", func(t *testing.T) {
			c, _, _ := setup4TestDesignationController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DesignationPath+"?season=abc", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestDesignationController(t)

			mockUsecase.EXPECT().GetByUserId(context.Background(), uid, "2026").Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DesignationPath+"?season=2026", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetRankStats", func(t *testing.T) {
		t.Run("正常系_season指定でティア別の分布を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestDesignationController(t)

			view := &usecase.DesignationRankStatsView{}

			mockUsecase.EXPECT().GetRankStats(context.Background(), "2026").Return(view, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DesignationsPath+DesignationStatsPath+"?season=2026", nil)
			c.router.ServeHTTP(w, req)

			var res dto.DesignationRankStatsResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "2026", res.Season)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestDesignationController(t)

			mockUsecase.EXPECT().GetRankStats(context.Background(), "2026").Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DesignationsPath+DesignationStatsPath+"?season=2026", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
