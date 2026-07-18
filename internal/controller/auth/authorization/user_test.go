package authorization

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
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

	t.Run("正常系_本人なら通過する", func(t *testing.T) {
		// パスパラメータで使用するid
		id := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// authenticationでセットされるuid
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// authentication後に実行されるMiddlewareのため、uidをセットしておく
		helper.SetUID(ginContext, uid)

		// パスパラメータにidが必要なMiddlewareのため、パスパラメータを追加
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

		middleware := UserAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	// uidが空の場合のテスト
	t.Run("異常系_未認証なら403を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		middleware := UserAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	// idに対応するユーザーが存在しない場合のテスト
	t.Run("異常系_ユーザが存在しなければ404を返す", func(t *testing.T) {
		// パスパラメータで使用するid
		id := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// authenticationでセットされるuid
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// authentication後に実行されるMiddlewareのため、uidをセットしておく
		helper.SetUID(ginContext, uid)

		// パスパラメータにidが必要なMiddlewareのため、パスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		middleware := UserAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	// idに対応するユーザーの取得に失敗した場合のテスト
	t.Run("異常系_取得エラーなら500を返す", func(t *testing.T) {
		// パスパラメータで使用するid
		id := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// authenticationでセットされるuid
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// authentication後に実行されるMiddlewareのため、uidをセットしておく
		helper.SetUID(ginContext, uid)

		// パスパラメータにidが必要なMiddlewareのため、パスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		middleware := UserAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 認証されたユーザーがidに対応するユーザーと異なる場合のテスト
	t.Run("異常系_他人のユーザIDなら403を返す", func(t *testing.T) {
		// パスパラメータで使用するid
		id := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// authenticationでセットされるuid
		// uidをidと異なる値にする
		uid, err := generateId()
		require.NoError(t, err)

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// authentication後に実行されるMiddlewareのため、uidをセットしておく
		helper.SetUID(ginContext, uid)

		// パスパラメータにidが必要なMiddlewareのため、パスパラメータを追加
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

		middleware := UserAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}
