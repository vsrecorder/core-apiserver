package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func setupMock4TestDeckController(t *testing.T) (
	*mock_repository.MockDeckInterface,
	*mock_repository.MockRecordInterface,
	*mock_usecase.MockDeckInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockDeckRepository := mock_repository.NewMockDeckInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	mockUsecase := mock_usecase.NewMockDeckInterface(mockCtrl)

	return mockDeckRepository, mockRecordRepository, mockUsecase
}

func setup4TestDeckController(t *testing.T, r *gin.Engine) (
	*Deck,
	*mock_repository.MockDeckInterface,
	*mock_repository.MockRecordInterface,
	*mock_usecase.MockDeckInterface,
) {
	mockDeckRepository, mockRecordRepository, mockUsecase := setupMock4TestDeckController(t)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := NewDeck(logger, r, mockDeckRepository, mockRecordRepository, mockUsecase)
	c.RegisterRoute("")

	return c, mockDeckRepository, mockRecordRepository, mockUsecase
}

func encodeTestDeckCursor(createdAt time.Time) string {
	return base64.StdEncoding.EncodeToString([]byte(createdAt.Format(time.RFC3339)))
}

// cursorEq はカーソルの時刻を検証するgomockマッチャ。
// カーソルはRFC3339の文字列を経由して復元されるため、モノトニッククロックや
// Locationのポインタがリクエスト前の値と一致せず、gomock既定のDeepEqualでは
// 一致しない。時刻そのものが等しいかをtime.Equalで見る。
func cursorEq(expected time.Time) gomock.Matcher {
	return gomock.Cond(func(actual time.Time) bool {
		return actual.Equal(expected)
	})
}

// newTestDeck はレスポンス生成に必要な値(LatestDeckCodeは必須)を埋めたDeckを返す。
func newTestDeck(id string, uid string, deckCode string, privateCodeFlg bool) *entity.Deck {
	return &entity.Deck{
		ID:         id,
		CreatedAt:  time.Now().Local(),
		UserId:     uid,
		Name:       "テストデッキ",
		PrivateFlg: false,
		LatestDeckCode: &entity.DeckCode{
			ID:             "01HD7Y3K8D6FDHMHTZ2GT41TC1",
			UserId:         uid,
			DeckId:         id,
			Code:           deckCode,
			PrivateCodeFlg: privateCodeFlg,
		},
	}
}

func TestDeckController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"Get":         test_DeckController_Get,
		"GetAll":      test_DeckController_GetAll,
		"GetById":     test_DeckController_GetById,
		"GetByUserId": test_DeckController_GetByUserId,
		"Create":      test_DeckController_Create,
		"Update":      test_DeckController_Update,
		"Archive":     test_DeckController_Archive,
		"Unarchive":   test_DeckController_Unarchive,
		"Delete":      test_DeckController_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_DeckController_Get(t *testing.T) {
	r := gin.Default()
	c, _, _, mockUsecase := setup4TestDeckController(t, r)

	// 未認証の場合、公開デッキの一覧が返る
	t.Run("正常系_未認証なら公開デッキ一覧を返す", func(t *testing.T) {
		limit := 10
		offset := 0

		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", "zor5SLfEfwfZ90yRVXzlxBEFARy2", "5dbFbk-uBwjqP-VVk5Vv", false),
		}

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
		require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.Decks[0].Data.LatestDeckCode.Code)
	})

	// 非公開のデッキコードは、未認証のユーザには伏せられる
	t.Run("正常系_非公開デッキコードは未認証には伏せられる", func(t *testing.T) {
		limit := 10
		offset := 0

		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", "zor5SLfEfwfZ90yRVXzlxBEFARy2", "5dbFbk-uBwjqP-VVk5Vv", true),
		}

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, res.Decks[0].Data.LatestDeckCode.Code)
	})

	// カーソルが指定された場合はFindOnCursorが使われる
	t.Run("正常系_カーソル指定時はFindOnCursorで取得する", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local().Truncate(time.Second)

		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", "zor5SLfEfwfZ90yRVXzlxBEFARy2", "", false),
		}

		mockUsecase.EXPECT().FindOnCursor(context.Background(), limit, cursorEq(cursor)).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&cursor=%s", limit, encodeTestDeckCursor(cursor)), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, len(decks), len(res.Decks))
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		limit := 10
		offset := 0

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 不正なlimitはバリデーションで弾かれる
	t.Run("異常系_limitが不正なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"?limit=invalid", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 不正なカーソルはバリデーションで弾かれる
	t.Run("異常系_カーソルが不正なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"?cursor=invalid", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_DeckController_GetAll(t *testing.T) {
	r := gin.Default()

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	c, _, _, mockUsecase := setup4TestDeckController(t, r)

	t.Run("正常系_自分の全デッキを返す", func(t *testing.T) {
		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", uid, "5dbFbk-uBwjqP-VVk5Vv", false),
		}

		mockUsecase.EXPECT().FindAll(context.Background(), uid).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/all", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetAllResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, len(decks), len(res))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", res[0].ID)
	})

	// 認証が必須のエンドポイント
	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/all", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		mockUsecase.EXPECT().FindAll(context.Background(), uid).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/all", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_GetById(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("正常系_指定IDのデッキを返す", func(t *testing.T) {
		r := gin.Default()
		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "5dbFbk-uBwjqP-VVk5Vv", false)

		// DeckGetByIdAuthorizationMiddlewareが公開設定の確認のために参照する
		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockUsecase.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.LatestDeckCode.Code)
	})

	// 非公開のデッキコードは、他人には伏せられる
	t.Run("正常系_非公開デッキコードは他人には伏せられる", func(t *testing.T) {
		r := gin.Default()
		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "5dbFbk-uBwjqP-VVk5Vv", true)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockUsecase.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, res.LatestDeckCode.Code)
	})

	// 非公開のデッキコードでも、本人であれば参照できる
	t.Run("正常系_非公開デッキコードでも本人には見える", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "5dbFbk-uBwjqP-VVk5Vv", true)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockUsecase.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.LatestDeckCode.Code)
	})

	// 非公開のデッキは他人からは参照できない
	t.Run("異常系_非公開デッキは他人からは403", func(t *testing.T) {
		r := gin.Default()
		c, mockRepository, _, _ := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "", false)
		deck.PrivateFlg = true

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_存在しないデッキは404を返す", func(t *testing.T) {
		r := gin.Default()
		c, mockRepository, _, _ := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()
		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_GetByUserId(t *testing.T) {
	r := gin.Default()

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	c, _, _, mockUsecase := setup4TestDeckController(t, r)

	// 認証済みの場合、同じGETでも自分のデッキ一覧が返る
	t.Run("正常系_認証済みなら自分のデッキ一覧を返す", func(t *testing.T) {
		limit := 10
		offset := 0
		archived := false

		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", uid, "5dbFbk-uBwjqP-VVk5Vv", true),
		}

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, archived, limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.False(t, res.Archived)
		require.Equal(t, len(decks), len(res.Decks))
		// 本人のデッキコードは非公開設定でも伏せられない
		require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", res.Decks[0].Data.LatestDeckCode.Code)
	})

	// archived=true の場合、アーカイブ済みのデッキ一覧が返る
	t.Run("正常系_archived指定でアーカイブ済み一覧を返す", func(t *testing.T) {
		limit := 10
		offset := 0
		archived := true

		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", uid, "", false),
		}
		decks[0].ArchivedAt = time.Now().Local()

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, archived, limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&offset=%d&archived=true", limit, offset), nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.True(t, res.Archived)
		require.NotEmpty(t, res.Decks[0].Data.ArchivedAt)
	})

	// カーソルが指定された場合はFindByUserIdOnCursorが使われる
	t.Run("正常系_カーソル指定時はFindByUserIdOnCursorで取得する", func(t *testing.T) {
		limit := 10
		archived := false
		cursor := time.Now().Local().Truncate(time.Second)

		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", uid, "", false),
		}

		mockUsecase.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archived, limit, cursorEq(cursor)).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&cursor=%s", limit, encodeTestDeckCursor(cursor)), nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, len(decks), len(res.Decks))
	})

	// 他人の非公開のデッキコードは伏せられる
	t.Run("正常系_他人の非公開デッキコードは伏せられる", func(t *testing.T) {
		limit := 10
		offset := 0
		archived := false

		decks := []*entity.Deck{
			newTestDeck("01HD7Y3K8D6FDHMHTZ2GT41TN2", "CeQ0Oa9g9uRThL11lj4l45VAg8p1", "5dbFbk-uBwjqP-VVk5Vv", true),
		}

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, archived, limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, res.Decks[0].Data.LatestDeckCode.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		limit := 10
		offset := 0
		archived := false

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, archived, limit, offset).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(DecksPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 不正なarchivedはバリデーションで弾かれる
	t.Run("異常系_archivedが不正なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", DecksPath+"?archived=invalid", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func test_DeckController_Create(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	// デッキコードを指定する場合、バリデーションが公式サイトへ問い合わせるため、
	// ここではデッキコードなしのリクエストのみを扱う
	t.Run("正常系_デッキコードなしのデッキを作成する", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, _, mockUsecase := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "", false)

		param := usecase.NewDeckCreateParam(
			uid,
			"テストデッキ",
			false,
			"",
			false,
			nil,
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(deck, nil)

		data := dto.DeckCreateRequest{
			Name:       "テストデッキ",
			PrivateFlg: false,
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", DecksPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, uid, res.UserId)
	})

	// ポケモンスプライトはリクエストからパラメータへ引き継がれる
	t.Run("正常系_ポケモンスプライトがパラメータへ引き継がれる", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, _, mockUsecase := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "", false)
		deck.PokemonSprites = []*entity.PokemonSprite{entity.NewPokemonSprite("pikachu")}

		param := usecase.NewDeckCreateParam(
			uid,
			"テストデッキ",
			true,
			"",
			false,
			[]*usecase.PokemonSpriteParam{usecase.NewPokemonSpriteParam("pikachu")},
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(deck, nil)

		data := dto.DeckCreateRequest{
			Name:           "テストデッキ",
			PrivateFlg:     true,
			PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "pikachu"}},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", DecksPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Len(t, res.PokemonSprites, 1)
		require.Equal(t, "pikachu", res.PokemonSprites[0].ID)
	})

	// デッキ名は必須
	t.Run("異常系_デッキ名が空なら400を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, _, _ := setup4TestDeckController(t, r)

		dataBytes, err := json.Marshal(dto.DeckCreateRequest{Name: ""})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", DecksPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 認証が必須のエンドポイント
	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, _, _ := setup4TestDeckController(t, r)

		dataBytes, err := json.Marshal(dto.DeckCreateRequest{Name: "テストデッキ"})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", DecksPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, _, mockUsecase := setup4TestDeckController(t, r)

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, errors.New(""))

		dataBytes, err := json.Marshal(dto.DeckCreateRequest{Name: "テストデッキ"})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", DecksPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// デッキコードの取得元がメンテナンス中の場合は503を返す
	t.Run("異常系_公式サイトメンテナンス中は503を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, _, mockUsecase := setup4TestDeckController(t, r)

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, apperror.ErrUnderMaintenance)

		dataBytes, err := json.Marshal(dto.DeckCreateRequest{Name: "テストデッキ"})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", DecksPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func test_DeckController_Update(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("正常系_本人のデッキを更新する", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "", false)

		// DeckUpdateAuthorizationMiddlewareが本人確認のために参照する
		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		param := usecase.NewDeckUpdateParam(
			"更新後のデッキ",
			true,
			[]*usecase.PokemonSpriteParam{usecase.NewPokemonSpriteParam("pikachu")},
		)

		mockUsecase.EXPECT().Update(context.Background(), id, param).Return(deck, nil)

		data := dto.DeckUpdateRequest{
			Name:           "更新後のデッキ",
			PrivateFlg:     true,
			PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "pikachu"}},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", DecksPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
	})

	// 他人のデッキは更新できない
	t.Run("異常系_他人のデッキは403を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, _ := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(
			newTestDeck(id, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", "", false), nil,
		)

		dataBytes, err := json.Marshal(dto.DeckUpdateRequest{Name: "更新後のデッキ"})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", DecksPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	// デッキ名は必須
	t.Run("異常系_デッキ名が空なら400を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, _ := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)

		dataBytes, err := json.Marshal(dto.DeckUpdateRequest{Name: ""})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", DecksPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockUsecase.EXPECT().Update(context.Background(), id, gomock.Any()).Return(nil, errors.New(""))

		dataBytes, err := json.Marshal(dto.DeckUpdateRequest{Name: "更新後のデッキ"})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", DecksPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Archive(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("正常系_本人のデッキをアーカイブする", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		deck := newTestDeck(id, uid, "", false)
		archivedDeck := newTestDeck(id, uid, "", false)
		archivedDeck.ArchivedAt = time.Now().Local()

		// DeckArchiveAuthorizationMiddlewareが本人確認のために参照する
		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockUsecase.EXPECT().Archive(context.Background(), id).Return(archivedDeck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", DecksPath+"/"+id+"/archive", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckArchiveResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.NotEmpty(t, res.ArchivedAt)
	})

	// 他人のデッキはアーカイブできない
	t.Run("異常系_他人のデッキは403を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, _ := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(
			newTestDeck(id, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", "", false), nil,
		)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", DecksPath+"/"+id+"/archive", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockUsecase.EXPECT().Archive(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", DecksPath+"/"+id+"/archive", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Unarchive(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("正常系_本人のデッキをアーカイブ解除する", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		archivedDeck := newTestDeck(id, uid, "", false)
		archivedDeck.ArchivedAt = time.Now().Local()

		mockRepository.EXPECT().FindById(context.Background(), id).Return(archivedDeck, nil)
		mockUsecase.EXPECT().Unarchive(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", DecksPath+"/"+id+"/unarchive", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.DeckUnarchiveResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, res.ArchivedAt)
	})

	t.Run("異常系_他人のデッキは403を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, _ := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(
			newTestDeck(id, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", "", false), nil,
		)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", DecksPath+"/"+id+"/unarchive", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockRepository, _, mockUsecase := setup4TestDeckController(t, r)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockUsecase.EXPECT().Unarchive(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", DecksPath+"/"+id+"/unarchive", nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Delete(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("正常系_記録に未使用の本人デッキを削除する", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockDeckRepository, mockRecordRepository, mockUsecase := setup4TestDeckController(t, r)

		// DeckDeleteAuthorizationMiddlewareが本人確認と、
		// 紐づく対戦記録が無いことの確認のために参照する
		mockDeckRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockRecordRepository.EXPECT().FindByDeckId(context.Background(), id, 1, 0, "").Return([]*entity.Record{}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", DecksPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	// 対戦記録に使われているデッキは削除できない
	t.Run("異常系_記録に使用中のデッキは409を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockDeckRepository, mockRecordRepository, _ := setup4TestDeckController(t, r)

		mockDeckRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockRecordRepository.EXPECT().FindByDeckId(context.Background(), id, 1, 0, "").Return(
			[]*entity.Record{{ID: "01HD7Y3K8D6FDHMHTZ2GT41TR1"}}, nil,
		)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", DecksPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusConflict, w.Code)
	})

	// 他人のデッキは削除できない
	t.Run("異常系_他人のデッキは403を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockDeckRepository, _, _ := setup4TestDeckController(t, r)

		mockDeckRepository.EXPECT().FindById(context.Background(), id).Return(
			newTestDeck(id, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", "", false), nil,
		)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", DecksPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockDeckRepository, mockRecordRepository, mockUsecase := setup4TestDeckController(t, r)

		mockDeckRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockRecordRepository.EXPECT().FindByDeckId(context.Background(), id, 1, 0, "").Return([]*entity.Record{}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", DecksPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 認可を通過した後に削除対象が消えていた場合(競合)は、404ではなく400を返す
	t.Run("異常系_認可後に消えていた場合は400を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockDeckRepository, mockRecordRepository, mockUsecase := setup4TestDeckController(t, r)

		mockDeckRepository.EXPECT().FindById(context.Background(), id).Return(newTestDeck(id, uid, "", false), nil)
		mockRecordRepository.EXPECT().FindByDeckId(context.Background(), id, 1, 0, "").Return([]*entity.Record{}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", DecksPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 存在しないデッキは認可の段階で404になる
	t.Run("異常系_存在しないデッキは認可段階で404を返す", func(t *testing.T) {
		r := gin.Default()

		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockDeckRepository, _, _ := setup4TestDeckController(t, r)

		mockDeckRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", DecksPath+"/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})
}
