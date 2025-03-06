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

func setupMock4TestUserController(t *testing.T) *mock_usecase.MockUserInterface {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockUserInterface(mockCtrl)

	return mockUsecase
}

func setup4TestUserController(t *testing.T, r *gin.Engine) (
	*User,
	*mock_usecase.MockUserInterface,
) {
	mockUsecase := setupMock4TestUserController(t)

	c := NewUser(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestUserController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetById": test_UserController_GetById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_UserController_GetById(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestUserController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()

		user := &entity.User{
			ID:          id,
			DisplayName: "",
			PhotoURL:    "",
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(user, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		var res dto.UserGetByIdResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
