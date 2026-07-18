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

func TestRecordAuthorizationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"RecordAuthorizationMiddleware":        test_RecordAuthorizationMiddleware,
		"RecordGetByIdAuthorizationMiddleware": test_RecordGetByIdAuthorizationMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_RecordAuthorizationMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	t.Run("正常系_所有者なら通過する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		record := &entity.Record{
			ID:     id,
			UserId: uid,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_未認証なら403を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_記録が存在しなければ404を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_取得エラーなら500を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_他人の記録なら403を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		record := &entity.Record{
			ID:     id,
			UserId: "KBp7roRDZobZg1t0OPzFR1kvLeO2",
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}

func test_RecordGetByIdAuthorizationMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	t.Run("正常系_所有者なら通過する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		record := &entity.Record{
			ID:     id,
			UserId: uid,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetByIdAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("正常系_公開記録なら未認証でも通過する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		record := &entity.Record{
			ID:     id,
			UserId: uid,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetByIdAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_id未指定で該当記録がなければ404を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		mockRepository.EXPECT().FindById(context.Background(), "").Return(nil, apperror.ErrRecordNotFound)

		middleware := RecordGetByIdAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_記録が存在しなければ404を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetByIdAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_取得エラーなら500を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetByIdAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_非公開の他人の記録は未認証なら403を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		record := &entity.Record{
			ID:         id,
			UserId:     "KBp7roRDZobZg1t0OPzFR1kvLeO2",
			PrivateFlg: true,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetByIdAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_非公開の他人の記録は認証済みでも403を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		id, err := generateId()
		require.NoError(t, err)

		// Middlewareのテストのためuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		// idが必要なMiddlewareのテストのためパスパラメータを追加
		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: id,
			},
		)

		record := &entity.Record{
			ID:         id,
			UserId:     "KBp7roRDZobZg1t0OPzFR1kvLeO2",
			PrivateFlg: true,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := RecordGetByIdAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}
