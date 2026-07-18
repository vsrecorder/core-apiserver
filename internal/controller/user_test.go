package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

var l = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// setJWTAuthHeader はテスト用に有効なJWTを生成し、Authorizationヘッダーへ付与する。
func setJWTAuthHeader(t *testing.T, req *http.Request, uid string, secretKey string) {
	token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
}

func setupMock4TestUserController(t *testing.T) (*mock_repository.MockUserInterface, *mock_usecase.MockUserInterface) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserInterface(mockCtrl)
	mockUsecase := mock_usecase.NewMockUserInterface(mockCtrl)

	return mockRepository, mockUsecase
}

func setup4TestUserController(t *testing.T, l *slog.Logger, r *gin.Engine) (
	*User,
	*mock_repository.MockUserInterface,
	*mock_usecase.MockUserInterface,
) {
	mockRepository, mockUsecase := setupMock4TestUserController(t)

	c := NewUser(l, r, mockRepository, mockUsecase)
	c.RegisterRoute("")

	return c, mockRepository, mockUsecase
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

	c, _, mockUsecase := setup4TestUserController(t, l, r)

	t.Run("正常系_指定IDのユーザを返す", func(t *testing.T) {
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

	t.Run("異常系_ユーザが存在しなければ404を返す", func(t *testing.T) {
		id, _ := generateId()

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		id, _ := generateId()

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_UserController_Create(t *testing.T) {
	t.Run("正常系_認証済みユーザを作成する", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためJWTを生成
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, mockUsecase := setup4TestUserController(t, l, r)

		name := "test"
		imageURL := "https://example.com/image.png"

		createdAt := time.Now().Local()

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
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.UserCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, name, res.Name)
		require.Equal(t, imageURL, res.ImageURL)
	})

	t.Run("異常系_既存ユーザなら409を返す", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためJWTを生成
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, mockUsecase := setup4TestUserController(t, l, r)

		name := "test"
		imageURL := "https://example.com/image.png"

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, apperror.ErrAlreadyExists)

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
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためJWTを生成
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, mockUsecase := setup4TestUserController(t, l, r)

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
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_UserController_Update(t *testing.T) {
	t.Run("正常系_本人の情報を更新する", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためJWTを生成
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, mockUsecase := setup4TestUserController(t, l, r)

		// UserUpdateAuthorizationMiddlewareが本人確認のために参照する
		mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.User{ID: id}, nil)

		name := "test"
		imageURL := "https://example.com/image.png"

		createdAt := time.Now().Local()

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
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.UserUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, name, res.Name)
		require.Equal(t, imageURL, res.ImageURL)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()
		id, _ := generateId()

		// 認証済みとするためJWTを生成
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, mockUsecase := setup4TestUserController(t, l, r)

		// UserUpdateAuthorizationMiddlewareが本人確認のために参照する
		mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.User{ID: id}, nil)

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
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_UserController_Delete(t *testing.T) {
	r := gin.Default()
	c, mockRepository, mockUsecase := setup4TestUserController(t, l, r)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	t.Run("正常系_本人のアカウントを削除する", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		// UserDeleteAuthorizationMiddlewareが本人確認のために参照する
		mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.User{ID: id}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", UsersPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_削除対象が存在しなければ400を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.User{ID: id}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", UsersPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.User{ID: id}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", UsersPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, id, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
