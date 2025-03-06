package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

const (
	DateLayout = "2006-01-02"
)

func setupMock4TestOfficialEventController(t *testing.T) *mock_usecase.MockOfficialEventInterface {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockOfficialEventInterface(mockCtrl)

	return mockUsecase
}

func setup4TestOfficialEventController(t *testing.T, r *gin.Engine) (
	*OfficialEvent,
	*mock_usecase.MockOfficialEventInterface,
) {
	mockUsecase := setupMock4TestOfficialEventController(t)

	c := NewOfficialEvent(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestOfficialEventController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"Get":     test_OfficialEventController_Get,
		"GetById": test_OfficialEventController_GetById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_OfficialEventController_Get(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestOfficialEventController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		officialEvents := []*entity.OfficialEvent{
			{
				ID: uint(606466),
			},
			{
				ID: uint(630879),
			},
		}

		typeId := uint(0)
		leagueType := uint(0)
		now := time.Now().UTC().Truncate(0)
		startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		mockUsecase.EXPECT().Find(context.Background(), typeId, leagueType, startDate, endDate).Return(officialEvents, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", OfficialEventsPath, nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, typeId, res.TypeId)
		require.Equal(t, leagueType, res.LeagueType)
		require.Equal(t, startDate, res.StartDate)
		require.Equal(t, endDate, res.EndDate)
		require.Equal(t, uint(606466), res.OfficialEvents[0].ID)
		require.Equal(t, uint(630879), res.OfficialEvents[1].ID)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		officialEvents := []*entity.OfficialEvent{
			{
				ID: uint(606466),
			},
			{
				ID: uint(630879),
			},
		}

		typeId := uint(1)
		leagueType := uint(0)
		now := time.Now().UTC().Truncate(0)
		startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		mockUsecase.EXPECT().Find(context.Background(), typeId, leagueType, startDate, endDate).Return(officialEvents, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf(OfficialEventsPath+"?type_id=%d", typeId), nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, typeId, res.TypeId)
		require.Equal(t, leagueType, res.LeagueType)
		require.Equal(t, startDate, res.StartDate)
		require.Equal(t, endDate, res.EndDate)
		require.Equal(t, uint(606466), res.OfficialEvents[0].ID)
		require.Equal(t, uint(630879), res.OfficialEvents[1].ID)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		officialEvents := []*entity.OfficialEvent{
			{
				ID: uint(606466),
			},
			{
				ID: uint(630879),
			},
		}

		typeId := uint(1)
		leagueType := uint(4)
		now := time.Now().UTC().Truncate(0)
		startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		mockUsecase.EXPECT().Find(context.Background(), typeId, leagueType, startDate, endDate).Return(officialEvents, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf(OfficialEventsPath+"?type_id=%d&league_type=%d", typeId, leagueType), nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, typeId, res.TypeId)
		require.Equal(t, leagueType, res.LeagueType)
		require.Equal(t, startDate, res.StartDate)
		require.Equal(t, endDate, res.EndDate)
		require.Equal(t, uint(606466), res.OfficialEvents[0].ID)
		require.Equal(t, uint(630879), res.OfficialEvents[1].ID)
	})

	t.Run("正常系_#04", func(t *testing.T) {
		officialEvents := []*entity.OfficialEvent{
			{
				ID: uint(606466),
			},
			{
				ID: uint(630879),
			},
		}

		typeId := uint(1)
		leagueType := uint(4)
		startDate := "2025-02-15"
		endDate := "2025-02-15"
		expectedStartDate, _ := time.Parse(DateLayout, startDate)
		expectedStartDate = time.Date(expectedStartDate.Year(), expectedStartDate.Month(), expectedStartDate.Day(), 0, 0, 0, 0, time.UTC)
		expectedEndDate, _ := time.Parse(DateLayout, endDate)
		expectedEndDate = time.Date(expectedEndDate.Year(), expectedEndDate.Month(), expectedEndDate.Day(), 0, 0, 0, 0, time.UTC)

		mockUsecase.EXPECT().Find(context.Background(), typeId, leagueType, expectedStartDate, expectedEndDate).Return(officialEvents, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf(OfficialEventsPath+"?type_id=%d&league_type=%d&start_date=%s&end_date=%s", typeId, leagueType, startDate, endDate), nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, typeId, res.TypeId)
		require.Equal(t, leagueType, res.LeagueType)
		require.Equal(t, expectedStartDate, res.StartDate)
		require.Equal(t, expectedEndDate, res.EndDate)
		require.Equal(t, uint(606466), res.OfficialEvents[0].ID)
		require.Equal(t, uint(630879), res.OfficialEvents[1].ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		typeId := uint(0)
		leagueType := uint(0)
		now := time.Now().UTC().Truncate(0)
		startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		mockUsecase.EXPECT().Find(context.Background(), typeId, leagueType, startDate, endDate).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", OfficialEventsPath, nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_OfficialEventController_GetById(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestOfficialEventController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id := uint(606466)

		officialEvent := &entity.OfficialEvent{
			ID: id,
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(officialEvent, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf(OfficialEventsPath+"/%d", id), nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetByIdResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id := uint(606466)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf(OfficialEventsPath+"/%d", id), nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetByIdResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id := uint(606466)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf(OfficialEventsPath+"/%d", id), nil)
		c.router.ServeHTTP(w, req)

		var res dto.OfficialEventGetByIdResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
