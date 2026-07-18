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

func newDeckCodeAuthContext(t *testing.T, id string, uid string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	if uid != "" {
		helper.SetUID(ctx, uid)
	}

	ctx.Params = append(ctx.Params, gin.Param{Key: "id", Value: id})

	return ctx, w
}

func TestDeckCodeAuthorizationMiddleware(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_所有者なら通過する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)

		DeckCodeAuthorizationMiddleware(mockRepository)(ctx)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_未認証なら403を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, "")

		DeckCodeAuthorizationMiddleware(mockRepository)(ctx)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_デッキコードが存在しなければ404を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		DeckCodeAuthorizationMiddleware(mockRepository)(ctx)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_取得エラーなら500を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		DeckCodeAuthorizationMiddleware(mockRepository)(ctx)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_他人のデッキコードなら403を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(
			&entity.DeckCode{ID: id, UserId: "KBp7roRDZobZg1t0OPzFR1kvLeO2"}, nil,
		)

		DeckCodeAuthorizationMiddleware(mockRepository)(ctx)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestDeckCodeDeleteAuthorizationMiddleware(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	setup := func(t *testing.T) (*mock_repository.MockDeckCodeInterface, *mock_repository.MockRecordInterface) {
		mockCtrl := gomock.NewController(t)
		return mock_repository.NewMockDeckCodeInterface(mockCtrl), mock_repository.NewMockRecordInterface(mockCtrl)
	}

	t.Run("正常系_記録に未使用の本人デッキコードなら通過する", func(t *testing.T) {
		mockDeckCodeRepository, mockRecordRepository := setup(t)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)
		mockRecordRepository.EXPECT().FindByDeckCodeId(context.Background(), id, 1, 0).Return([]*entity.Record{}, nil)

		DeckCodeDeleteAuthorizationMiddleware(mockDeckCodeRepository, mockRecordRepository)(ctx)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_記録に使用中のデッキコードは409を返す", func(t *testing.T) {
		mockDeckCodeRepository, mockRecordRepository := setup(t)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)
		mockRecordRepository.EXPECT().FindByDeckCodeId(context.Background(), id, 1, 0).Return(
			[]*entity.Record{{ID: "01HD7Y3K8D6FDHMHTZ2GT41TR1"}}, nil,
		)

		DeckCodeDeleteAuthorizationMiddleware(mockDeckCodeRepository, mockRecordRepository)(ctx)

		require.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("異常系_使用状況の取得エラーなら500を返す", func(t *testing.T) {
		mockDeckCodeRepository, mockRecordRepository := setup(t)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)
		mockRecordRepository.EXPECT().FindByDeckCodeId(context.Background(), id, 1, 0).Return(nil, errors.New(""))

		DeckCodeDeleteAuthorizationMiddleware(mockDeckCodeRepository, mockRecordRepository)(ctx)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_他人のデッキコードなら403を返す", func(t *testing.T) {
		mockDeckCodeRepository, mockRecordRepository := setup(t)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(
			&entity.DeckCode{ID: id, UserId: "KBp7roRDZobZg1t0OPzFR1kvLeO2"}, nil,
		)

		DeckCodeDeleteAuthorizationMiddleware(mockDeckCodeRepository, mockRecordRepository)(ctx)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_デッキコードが存在しなければ404を返す", func(t *testing.T) {
		mockDeckCodeRepository, mockRecordRepository := setup(t)

		id, err := generateId()
		require.NoError(t, err)

		ctx, w := newDeckCodeAuthContext(t, id, uid)

		mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		DeckCodeDeleteAuthorizationMiddleware(mockDeckCodeRepository, mockRecordRepository)(ctx)

		require.Equal(t, http.StatusNotFound, w.Code)
	})
}
