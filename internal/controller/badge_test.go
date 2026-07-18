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

func setup4TestBadgeController(t *testing.T) (
	*Badge,
	*mock_usecase.MockBadgeInterface,
	*mock_repository.MockChampionshipSeriesInterface,
) {
	gin.SetMode(gin.TestMode)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockBadgeInterface(mockCtrl)
	mockSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	r := gin.Default()
	c := NewBadge(r, mockUsecase, mockSeriesRepo)
	c.RegisterRoute("")

	return c, mockUsecase, mockSeriesRepo
}

func newTestBadgeDefinition(id string) *entity.BadgeDefinition {
	now := time.Now().Local()
	return entity.NewBadgeDefinition(
		id, "first_record", "onboarding", "はじめての記録", "初めて記録を作成した", "icon_first_record",
		"record_count", 1, now, time.Time{}, now, now,
	)
}

func TestBadgeController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("GetAllDefinitions", func(t *testing.T) {
		t.Run("正常系_バッジ定義一覧を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestBadgeController(t)

			definitions := []*entity.BadgeDefinition{newTestBadgeDefinition("badge-first-record")}

			mockUsecase.EXPECT().GetAllDefinitions(context.Background()).Return(definitions, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", BadgesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var res dto.BadgeDefinitionsResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			require.Len(t, res.Badges, 1)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestBadgeController(t)

			mockUsecase.EXPECT().GetAllDefinitions(context.Background()).Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", BadgesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetByUserId", func(t *testing.T) {
		t.Run("正常系_season指定でユーザの獲得状況を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestBadgeController(t)

			views := []*usecase.UserBadgeView{
				{Definition: newTestBadgeDefinition("badge-first-record"), Achieved: true, CurrentValue: 1},
			}

			mockUsecase.EXPECT().GetByUserId(context.Background(), uid, "2026").Return(views, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+BadgesPath+"?season=2026", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("正常系_season未指定なら現在のシーズンで判定する", func(t *testing.T) {
			c, mockUsecase, mockSeriesRepo := setup4TestBadgeController(t)

			// championship_seriesのIDから現在のシーズン識別子を解決する
			mockSeriesRepo.EXPECT().FindByDate(context.Background(), gomock.Any()).Return(
				&entity.ChampionshipSeries{ID: "series_2026"}, nil,
			)
			mockUsecase.EXPECT().GetByUserId(context.Background(), uid, "2026").Return([]*usecase.UserBadgeView{}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+BadgesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_seasonの形式が不正なら400を返す", func(t *testing.T) {
			c, _, _ := setup4TestBadgeController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+BadgesPath+"?season=abc", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_現在のシーズンが引けなければ500を返す", func(t *testing.T) {
			c, _, mockSeriesRepo := setup4TestBadgeController(t)

			mockSeriesRepo.EXPECT().FindByDate(context.Background(), gomock.Any()).Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+BadgesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, _ := setup4TestBadgeController(t)

			mockUsecase.EXPECT().GetByUserId(context.Background(), uid, "2026").Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+BadgesPath+"?season=2026", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
