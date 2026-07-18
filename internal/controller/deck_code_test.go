package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

// stubDeckCodeUsecase はデッキコードユースケースのスタブ。
// mock_usecaseにDeckCode用のモックが存在しないため手書きする。
type stubDeckCodeUsecase struct {
	deckCode  *entity.DeckCode
	deckCodes []*entity.DeckCode
	err       error
}

func (s stubDeckCodeUsecase) FindById(ctx context.Context, id string) (*entity.DeckCode, error) {
	return s.deckCode, s.err
}

func (s stubDeckCodeUsecase) FindByDeckId(ctx context.Context, deckId string) ([]*entity.DeckCode, error) {
	return s.deckCodes, s.err
}

func (s stubDeckCodeUsecase) Create(ctx context.Context, param *usecase.DeckCodeCreateParam) (*entity.DeckCode, error) {
	return s.deckCode, s.err
}

func (s stubDeckCodeUsecase) Update(ctx context.Context, id string, param *usecase.DeckCodeUpdateParam) (*entity.DeckCode, error) {
	return s.deckCode, s.err
}

func (s stubDeckCodeUsecase) Delete(ctx context.Context, id string) error {
	return s.err
}

func setup4TestDeckCodeController(t *testing.T, u stubDeckCodeUsecase) (
	*DeckCode,
	*mock_repository.MockDeckCodeInterface,
	*mock_repository.MockRecordInterface,
	string,
) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockDeckCodeRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := gin.Default()
	c := NewDeckCode(logger, r, mockDeckCodeRepository, mockRecordRepository, u)
	c.RegisterRoute("")

	return c, mockDeckCodeRepository, mockRecordRepository, secretKey
}

func newTestDeckCodeEntity(id string, uid string, privateCodeFlg bool) *entity.DeckCode {
	return entity.NewDeckCode(
		id, time.Now().Local(), uid, "01HD7Y3K8D6FDHMHTZ2GT41TD1", "5dbFbk-uBwjqP-VVk5Vv", privateCodeFlg, "メモ",
	)
}

func TestDeckCodeController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TC1"

	t.Run("GetById", func(t *testing.T) {
		t.Run("正常系_公開デッキコードは未認証でも参照できる", func(t *testing.T) {
			c, _, _, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, false)})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			var res dto.DeckCodeGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.Code)
		})

		t.Run("正常系_非公開デッキコードは他人には伏せられる", func(t *testing.T) {
			c, _, _, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, true)})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			var res dto.DeckCodeGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Empty(t, res.Code)
		})

		t.Run("正常系_非公開デッキコードでも本人には見える", func(t *testing.T) {
			c, _, _, secretKey := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, true)})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			var res dto.DeckCodeGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.Code)
		})

		t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
			c, _, _, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{err: apperror.ErrRecordNotFound})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("GetByDeckId", func(t *testing.T) {
		t.Run("正常系_他人の非公開デッキコードだけが伏せられる", func(t *testing.T) {
			deckCodes := []*entity.DeckCode{
				newTestDeckCodeEntity("01HD7Y3K8D6FDHMHTZ2GT41TC1", uid, true),
				newTestDeckCodeEntity("01HD7Y3K8D6FDHMHTZ2GT41TC2", "KBp7roRDZobZg1t0OPzFR1kvLeO2", true),
			}
			c, _, _, secretKey := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCodes: deckCodes})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/01HD7Y3K8D6FDHMHTZ2GT41TD1"+DeckCodesPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			var res []*dto.DeckCodeResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Len(t, res, 2)
			require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res[0].Code)
			require.Empty(t, res[1].Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, _, _, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{err: errors.New("")})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/01HD7Y3K8D6FDHMHTZ2GT41TD1"+DeckCodesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("Update", func(t *testing.T) {
		newRequestBody := func(t *testing.T) *http.Request {
			t.Helper()
			b, err := json.Marshal(dto.DeckCodeUpdateRequest{PrivateCodeFlg: true, Memo: "更新後のメモ"})
			require.NoError(t, err)
			req, err := http.NewRequest("PUT", DeckCodesPath+"/"+id, strings.NewReader(string(b)))
			require.NoError(t, err)
			return req
		}

		t.Run("正常系_本人のデッキコードを更新する", func(t *testing.T) {
			c, mockDeckCodeRepository, _, secretKey := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, true)})

			// DeckCodeUpdateAuthorizationMiddlewareが本人確認のために参照する
			mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)

			w := httptest.NewRecorder()
			req := newRequestBody(t)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_他人のデッキコードは403を返す", func(t *testing.T) {
			c, mockDeckCodeRepository, _, secretKey := setup4TestDeckCodeController(t, stubDeckCodeUsecase{})

			mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(
				&entity.DeckCode{ID: id, UserId: "KBp7roRDZobZg1t0OPzFR1kvLeO2"}, nil,
			)

			w := httptest.NewRecorder()
			req := newRequestBody(t)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_記録に未使用の本人デッキコードを削除する", func(t *testing.T) {
			c, mockDeckCodeRepository, mockRecordRepository, secretKey := setup4TestDeckCodeController(t, stubDeckCodeUsecase{})

			mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)
			mockRecordRepository.EXPECT().FindByDeckCodeId(context.Background(), id, 1, 0).Return([]*entity.Record{}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", DeckCodesPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_記録に使用中のデッキコードは409を返す", func(t *testing.T) {
			c, mockDeckCodeRepository, mockRecordRepository, secretKey := setup4TestDeckCodeController(t, stubDeckCodeUsecase{})

			mockDeckCodeRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)
			mockRecordRepository.EXPECT().FindByDeckCodeId(context.Background(), id, 1, 0).Return(
				[]*entity.Record{{ID: "01HD7Y3K8D6FDHMHTZ2GT41TR1"}}, nil,
			)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", DeckCodesPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusConflict, w.Code)
		})
	})
}
