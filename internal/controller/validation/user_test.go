package validation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestUserValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"UserCreateMiddleware": test_UserCreateMiddleware,
		"UserUpdateMiddleware": test_UserUpdateMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_UserCreateMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"
		imageURL := "https://example.com/image.jpg"

		expected := dto.UserCreateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetUserCreateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""
		imageURL := "https://example.com/image.jpg"

		expected := dto.UserCreateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"
		imageURL := ""

		expected := dto.UserCreateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""
		imageURL := ""

		expected := dto.UserCreateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_UserUpdateMiddleware(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"
		imageURL := "https://example.com/image.jpg"

		expected := dto.UserUpdateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserUpdateMiddleware()
		middleware(ginContext)

		actual := helper.GetUserUpdateRequest(ginContext)

		require.Equal(t, expected, actual)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""
		imageURL := "https://example.com/image.jpg"

		expected := dto.UserUpdateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := "test"
		imageURL := ""

		expected := dto.UserUpdateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#03", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		name := ""
		imageURL := ""

		expected := dto.UserUpdateRequest{
			UserRequest: dto.UserRequest{
				Name:     name,
				ImageURL: imageURL,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("PUT", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserUpdateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
