package authorization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"go.uber.org/mock/gomock"
)

func TestUserAuthorizationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"UserAuthorizationMiddleware": test_UserAuthorizationMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_UserAuthorizationMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserInterface(mockCtrl)

	// パスパラメータで使用するid
	id := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// authenticationでセットされるuid
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためuidをセット
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		user := &entity.User{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	// uidが空の場合のテスト
	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := UserAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}
