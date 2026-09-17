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
	*mock_repository.MockDeckInterface,
) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockDeckCodeRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)
	// 参照系の認可(親デッキの公開範囲)が引く
	mockDeckRepository := mock_repository.NewMockDeckInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := gin.Default()
	c := NewDeckCode(logger, r, mockDeckCodeRepository, mockDeckRepository, mockRecordRepository, u)
	c.RegisterRoute("")

	return c, mockDeckCodeRepository, mockRecordRepository, secretKey, mockDeckRepository
}

// testDeckCodeDeckId はテストのデッキコードが属するデッキのID。
const testDeckCodeDeckId = "01HD7Y3K8D6FDHMHTZ2GT41TD1"

func newTestDeckCodeEntity(id string, uid string, privateCodeFlg bool) *entity.DeckCode {
	return entity.NewDeckCode(
		id, time.Now().Local(), uid, testDeckCodeDeckId, "5dbFbk-uBwjqP-VVk5Vv", privateCodeFlg, "メモ",
	)
}

// newTestDeckForDeckCode はデッキコードの親デッキ(参照系の認可が公開範囲を見る)。
func newTestDeckForDeckCode(uid string, privateFlg bool) *entity.Deck {
	return &entity.Deck{ID: testDeckCodeDeckId, UserId: uid, PrivateFlg: privateFlg}
}

// expectDeckCodeReadAuthorization は GET /deckcodes/:id の認可ミドルウェアが引く、
// デッキコードと親デッキの取得を期待する。
func expectDeckCodeReadAuthorization(
	deckCodeRepo *mock_repository.MockDeckCodeInterface,
	deckRepo *mock_repository.MockDeckInterface,
	id string,
	uid string,
	privateDeck bool,
) {
	deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(newTestDeckCodeEntity(id, uid, false), nil)
	deckRepo.EXPECT().FindById(gomock.Any(), testDeckCodeDeckId).Return(newTestDeckForDeckCode(uid, privateDeck), nil)
}

func TestDeckCodeController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TC1"

	t.Run("GetById", func(t *testing.T) {
		t.Run("正常系_公開デッキコードは未認証でも参照できる", func(t *testing.T) {
			c, mockDeckCodeRepository, _, _, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, false)})
			expectDeckCodeReadAuthorization(mockDeckCodeRepository, mockDeckRepository, id, uid, false)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			var res dto.DeckCodeGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.Code)
		})

		t.Run("正常系_非公開デッキコードは他人には伏せられる", func(t *testing.T) {
			c, mockDeckCodeRepository, _, _, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, true)})
			expectDeckCodeReadAuthorization(mockDeckCodeRepository, mockDeckRepository, id, uid, false)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			var res dto.DeckCodeGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Empty(t, res.Code)
		})

		t.Run("正常系_非公開デッキコードでも本人には見える", func(t *testing.T) {
			c, mockDeckCodeRepository, _, secretKey, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, true)})
			expectDeckCodeReadAuthorization(mockDeckCodeRepository, mockDeckRepository, id, uid, false)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			var res dto.DeckCodeGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.Code)
		})

		// デッキ本体(GET /decks/:id)と同じく、非公開デッキの派生データは他人に見せない
		t.Run("異常系_非公開デッキのデッキコードは他人には403を返す", func(t *testing.T) {
			c, mockDeckCodeRepository, _, secretKey, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, false)})
			expectDeckCodeReadAuthorization(mockDeckCodeRepository, mockDeckRepository, id, uid, true)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
		})

		t.Run("正常系_非公開デッキのデッキコードでも所有者は参照できる", func(t *testing.T) {
			c, mockDeckCodeRepository, _, secretKey, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, false)})
			expectDeckCodeReadAuthorization(mockDeckCodeRepository, mockDeckRepository, id, uid, true)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodesPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
			c, mockDeckCodeRepository, _, _, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{err: apperror.ErrRecordNotFound})
			// 認可ミドルウェアの時点で見つからず、ユースケースへは進まない
			mockDeckCodeRepository.EXPECT().FindById(gomock.Any(), id).Return(nil, apperror.ErrRecordNotFound)

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
			c, _, _, secretKey, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCodes: deckCodes})
			mockDeckRepository.EXPECT().FindById(gomock.Any(), testDeckCodeDeckId).Return(newTestDeckForDeckCode(uid, false), nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/"+testDeckCodeDeckId+DeckCodesPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			var res []*dto.DeckCodeResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Len(t, res, 2)
			require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res[0].Code)
			require.Empty(t, res[1].Code)
		})

		t.Run("異常系_非公開デッキの一覧は他人には403を返す", func(t *testing.T) {
			c, _, _, secretKey, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCodes: []*entity.DeckCode{newTestDeckCodeEntity(id, uid, false)}})
			mockDeckRepository.EXPECT().FindById(gomock.Any(), testDeckCodeDeckId).Return(newTestDeckForDeckCode(uid, true), nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/"+testDeckCodeDeckId+DeckCodesPath, nil)
			setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
		})

		t.Run("異常系_非公開デッキの一覧は未認証にも403を返す", func(t *testing.T) {
			c, _, _, _, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCodes: []*entity.DeckCode{newTestDeckCodeEntity(id, uid, false)}})
			mockDeckRepository.EXPECT().FindById(gomock.Any(), testDeckCodeDeckId).Return(newTestDeckForDeckCode(uid, true), nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/"+testDeckCodeDeckId+DeckCodesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
		})

		t.Run("異常系_存在しないデッキは404を返す", func(t *testing.T) {
			c, _, _, _, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{})
			mockDeckRepository.EXPECT().FindById(gomock.Any(), testDeckCodeDeckId).Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/"+testDeckCodeDeckId+DeckCodesPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, _, _, _, mockDeckRepository := setup4TestDeckCodeController(t, stubDeckCodeUsecase{err: errors.New("")})
			mockDeckRepository.EXPECT().FindById(gomock.Any(), testDeckCodeDeckId).Return(newTestDeckForDeckCode(uid, false), nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/"+testDeckCodeDeckId+DeckCodesPath, nil)
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
			c, mockDeckCodeRepository, _, secretKey, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{deckCode: newTestDeckCodeEntity(id, uid, true)})

			// DeckCodeUpdateAuthorizationMiddlewareが本人確認のために参照する
			mockDeckCodeRepository.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)

			w := httptest.NewRecorder()
			req := newRequestBody(t)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_他人のデッキコードは403を返す", func(t *testing.T) {
			c, mockDeckCodeRepository, _, secretKey, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{})

			mockDeckCodeRepository.EXPECT().FindById(gomock.Any(), id).Return(
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
			c, mockDeckCodeRepository, mockRecordRepository, secretKey, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{})

			mockDeckCodeRepository.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)
			mockRecordRepository.EXPECT().FindByDeckCodeId(gomock.Any(), uid, id, 1, 0).Return([]*entity.Record{}, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", DeckCodesPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_記録に使用中のデッキコードは409を返す", func(t *testing.T) {
			c, mockDeckCodeRepository, mockRecordRepository, secretKey, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{})

			mockDeckCodeRepository.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)
			mockRecordRepository.EXPECT().FindByDeckCodeId(gomock.Any(), uid, id, 1, 0).Return(
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
