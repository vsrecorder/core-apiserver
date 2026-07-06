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

func TestMatchAuthorizationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"MatchAuthorizationMiddleware":        test_MatchAuthorizationMiddleware,
		"MatchReorderAuthorizationMiddleware": test_MatchReorderAuthorizationMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_MatchAuthorizationMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockMatchInterface(mockCtrl)

	t.Run("正常系_#01", func(t *testing.T) {
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

		match := &entity.Match{
			ID:     id,
			UserId: uid,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
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

		middleware := MatchAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#03", func(t *testing.T) {
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

		middleware := MatchAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_#04", func(t *testing.T) {
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

		match := &entity.Match{
			ID:     id,
			UserId: "KBp7roRDZobZg1t0OPzFR1kvLeO2",
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchAuthorizationMiddleware(mockRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}

func test_MatchReorderAuthorizationMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	t.Run("正常系_#01", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		recordId, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: recordId,
			},
		)

		record := &entity.Record{
			ID:     recordId,
			UserId: uid,
		}

		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(record, nil)

		req, err := http.NewRequest("PUT", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchReorderAuthorizationMiddleware(mockRecordRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_#01_recordが見つからない", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		recordId, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: recordId,
			},
		)

		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(nil, apperror.ErrRecordNotFound)

		req, err := http.NewRequest("PUT", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchReorderAuthorizationMiddleware(mockRecordRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02_他人のrecord", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		recordId, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: recordId,
			},
		)

		record := &entity.Record{
			ID:     recordId,
			UserId: "KBp7roRDZobZg1t0OPzFR1kvLeO2",
		}

		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(record, nil)

		req, err := http.NewRequest("PUT", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchReorderAuthorizationMiddleware(mockRecordRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_#03_privateでも所有者ならOK", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		recordId, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		helper.SetUID(ginContext, uid)

		ginContext.Params = append(
			ginContext.Params,
			gin.Param{
				Key:   "id",
				Value: recordId,
			},
		)

		record := &entity.Record{
			ID:         recordId,
			UserId:     uid,
			PrivateFlg: true,
		}

		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(record, nil)

		req, err := http.NewRequest("PUT", "/", nil)
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchReorderAuthorizationMiddleware(mockRecordRepository)
		middleware(ginContext)

		require.Equal(t, http.StatusOK, w.Code)
	})
}
