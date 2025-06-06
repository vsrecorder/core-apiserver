package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func setupMock4TestUserController(t *testing.T) (*mock_repository.MockUserInterface, *mock_usecase.MockUserInterface) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserInterface(mockCtrl)
	mockUsecase := mock_usecase.NewMockUserInterface(mockCtrl)

	return mockRepository, mockUsecase
}

func setup4TestUserController(t *testing.T, r *gin.Engine) (
	*User,
	*mock_usecase.MockUserInterface,
) {
	authDisable := true

	mockRepository, mockUsecase := setupMock4TestUserController(t)

	c := NewUser(r, mockRepository, mockUsecase)
	c.RegisterRoute("", authDisable)

	return c, mockUsecase
}

func TestUserController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetById": test_UserController_GetById,
		"Create":  test_UserController_Create,
		"Update":  test_UserController_Update,
		"Delete":  test_UserController_Delete,
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
			ID:       id,
			Name:     "",
			ImageURL: "",
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
		id, _ := generateId()

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, _ := generateId()

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_UserController_Create(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためにuidをセット
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, id)
		})

		c, mockUsecase := setup4TestUserController(t, r)

		name := "test"
		imageURL := "https://example.com/image.png"

		createdAt := time.Now().UTC().Truncate(0)

		user := &entity.User{
			ID:        id,
			CreatedAt: createdAt,
			Name:      name,
			ImageURL:  imageURL,
		}

		param := usecase.NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(user, nil)

		data := dto.UserCreateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", UsersPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.UserCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, name, res.Name)
		require.Equal(t, imageURL, res.ImageURL)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためにuidをセット
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, id)
		})

		c, mockUsecase := setup4TestUserController(t, r)

		name := "test"
		imageURL := "https://example.com/image.png"

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, ErrAlreadyExists)

		data := dto.UserCreateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", UsersPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためにuidをセット
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, id)
		})

		c, mockUsecase := setup4TestUserController(t, r)

		name := "test"
		imageURL := "https://example.com/image.png"

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, errors.New(""))

		data := dto.UserCreateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", UsersPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_UserController_Update(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためにuidをセット
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, id)
		})

		c, mockUsecase := setup4TestUserController(t, r)

		name := "test"
		imageURL := "https://example.com/image.png"

		createdAt := time.Now().UTC().Truncate(0)

		user := &entity.User{
			ID:        id,
			CreatedAt: createdAt,
			Name:      name,
			ImageURL:  imageURL,
		}

		param := usecase.NewUserUpdateParam(
			name,
			imageURL,
		)

		mockUsecase.EXPECT().Update(context.Background(), id, param).Return(user, nil)

		data := dto.UserUpdateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", UsersPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.UserUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, name, res.Name)
		require.Equal(t, imageURL, res.ImageURL)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためにuidをセット
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, id)
		})

		c, mockUsecase := setup4TestUserController(t, r)

		name := "test"
		imageURL := "https://example.com/image.png"

		mockUsecase.EXPECT().Update(context.Background(), id, gomock.Any()).Return(nil, errors.New(""))

		data := dto.UserUpdateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", UsersPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_UserController_Delete(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestUserController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", UsersPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", UsersPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", UsersPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
