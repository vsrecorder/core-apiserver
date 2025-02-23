package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"go.uber.org/mock/gomock"
)

func setupMock4TestTonamelEventController(t *testing.T) *mock_usecase.MockTonamelEventInterface {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockTonamelEventInterface(mockCtrl)

	return mockUsecase
}

func setup4TestTonamelEventController(t *testing.T, r *gin.Engine) (
	*TonamelEvent,
	*mock_usecase.MockTonamelEventInterface,
) {
	mockUsecase := setupMock4TestTonamelEventController(t)

	c := NewTonamelEvent(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestTonamelEventController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetById": test_TonamelEventController_GetById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_TonamelEventController_GetById(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestTonamelEventController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id := "61ozP"

		tonamelEvent := &entity.TonamelEvent{
			ID: id,
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(tonamelEvent, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", TonamelEventsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		var res dto.TonamelEventGetByIdResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id := "61ozP"

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", TonamelEventsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
