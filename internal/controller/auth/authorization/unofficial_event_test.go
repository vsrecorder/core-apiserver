package authorization

import (
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

func TestUnofficialEventAuthorizationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"UnofficialEventAuthorizationMiddleware": test_UnofficialEventAuthorizationMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_UnofficialEventAuthorizationMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUnofficialEventInterface(mockCtrl)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// Middlewareのテストのためuid・パスパラメータを持たせたgin.Contextを組み立てる
	setup := func(t *testing.T, withUID bool) (*gin.Context, *httptest.ResponseRecorder, string) {
		t.Helper()

		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		if withUID {
			helper.SetUID(ginContext, uid)
		}

		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		return ginContext, w, id
	}

	t.Run("正常系_所有者なら通過する", func(t *testing.T) {
		ginContext, w, id := setup(t, true)

		unofficialEvent := &entity.UnofficialEvent{
			ID:     id,
			UserId: uid,
		}

		mockRepository.EXPECT().FindById(gomock.Any(), id).Return(unofficialEvent, nil)

		UnofficialEventAuthorizationMiddleware(mockRepository)(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_未認証なら403を返す", func(t *testing.T) {
		ginContext, w, _ := setup(t, false)

		UnofficialEventAuthorizationMiddleware(mockRepository)(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_自由形式イベントが存在しなければ404を返す", func(t *testing.T) {
		ginContext, w, id := setup(t, true)

		mockRepository.EXPECT().FindById(gomock.Any(), id).Return(nil, apperror.ErrRecordNotFound)

		UnofficialEventAuthorizationMiddleware(mockRepository)(ginContext)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_取得エラーなら500を返す", func(t *testing.T) {
		ginContext, w, id := setup(t, true)

		mockRepository.EXPECT().FindById(gomock.Any(), id).Return(nil, errors.New(""))

		UnofficialEventAuthorizationMiddleware(mockRepository)(ginContext)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("正常系_削除用ミドルウェアも所有者なら通過する", func(t *testing.T) {
		ginContext, w, id := setup(t, true)

		unofficialEvent := &entity.UnofficialEvent{
			ID:     id,
			UserId: uid,
		}

		mockRepository.EXPECT().FindById(gomock.Any(), id).Return(unofficialEvent, nil)

		UnofficialEventDeleteAuthorizationMiddleware(mockRepository)(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_他人の自由形式イベントなら403を返す", func(t *testing.T) {
		ginContext, w, id := setup(t, true)

		unofficialEvent := &entity.UnofficialEvent{
			ID:     id,
			UserId: "KBp7roRDZobZg1t0OPzFR1kvLeO2",
		}

		mockRepository.EXPECT().FindById(gomock.Any(), id).Return(unofficialEvent, nil)

		UnofficialEventAuthorizationMiddleware(mockRepository)(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}
