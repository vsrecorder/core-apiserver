package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func setupMock4TestDeckController(t *testing.T) (*mock_repository.MockDeckInterface, *mock_usecase.MockDeckInterface) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckInterface(mockCtrl)
	mockUsecase := mock_usecase.NewMockDeckInterface(mockCtrl)

	return mockRepository, mockUsecase
}

func setup4TestDeckController(t *testing.T, r *gin.Engine) (
	*Deck,
	*mock_usecase.MockDeckInterface,
) {
	authDisable := true
	mockRepository, mockUsecase := setupMock4TestDeckController(t)

	c := NewDeck(r, mockRepository, mockUsecase)
	c.RegisterRoute("", authDisable)

	return c, mockUsecase
}

func TestDeckController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"Get":         test_DeckController_Get,
		"GetById":     test_DeckController_GetById,
		"GetByUserId": test_DeckController_GetByUserId,
		"Create":      test_DeckController_Create,
		"Update":      test_DeckController_Update,
		"Archive":     test_DeckController_Archive,
		"Unarchie":    test_DeckController_Unarchive,
		"Delete":      test_DeckController_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_DeckController_Get(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestDeckController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		deck := entity.Deck{}
		decks := []*entity.Deck{
			&deck,
		}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/decks?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
	})

	t.Run("正常系_#02", func(t *testing.T) {
		deck := entity.Deck{}
		decks := []*entity.Deck{
			&deck,
		}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
	})

	t.Run("正常系_#03", func(t *testing.T) {
		decks := []*entity.Deck{}

		limit := 10
		offset := 0
		cursor := base64.StdEncoding.EncodeToString([]byte(time.Time{}.Format(time.RFC3339)))

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"cursor\":\""+cursor+"\",\"decks\":[]}", w.Body.String())
	})

	t.Run("正常系_#04", func(t *testing.T) {
		deck := entity.Deck{}
		decks := []*entity.Deck{
			&deck,
		}

		limit := 10
		offset := 0
		cursor, err := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/decks?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
		require.Equal(t, base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))), res.Cursor)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		mockUsecase.EXPECT().Find(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		cursor, err := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindOnCursor(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/decks?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_GetById(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		createdAt := time.Now().UTC().Truncate(0)
		code := "01JGPC7829AMTNVVNX63VQF5XW"

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         uid,
			Name:           "",
			Code:           code,
			PrivateCodeFlg: false,
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, code, res.Code)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)
		code := "01JGPC7829AMTNVVNX63VQF5XW"

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         uid,
			Name:           "",
			Code:           code,
			PrivateCodeFlg: true,
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, code, res.Code)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         "zor5SLfEfwfZ90yRVXzlxBEFARy2",
			Name:           "",
			Code:           "01JGPC7829AMTNVVNX63VQF5XW",
			PrivateCodeFlg: true,
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Empty(t, res.Code)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/decks/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/decks/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_GetByUserId(t *testing.T) {
	r := gin.Default()

	// 認証済みとするためにuidをセット
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	r.Use(func(ctx *gin.Context) {
		helper.SetUID(ctx, uid)
	})

	c, mockUsecase := setup4TestDeckController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		deck := entity.Deck{
			UserId: uid,
		}

		decks := []*entity.Deck{
			&deck,
		}

		archived := false
		limit := 10
		offset := 0

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, archived, limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/decks?limit=%d&offset=%d&archived=%t", limit, offset, archived), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
		require.Equal(t, uid, res.Decks[0].Data.UserId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		deck := entity.Deck{
			UserId: uid,
		}

		decks := []*entity.Deck{
			&deck,
		}

		archived := false
		limit := 10
		offset := 0

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, archived, limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
		require.Equal(t, uid, res.Decks[0].Data.UserId)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		decks := []*entity.Deck{}

		archived := false
		limit := 10
		offset := 0
		cursor := base64.StdEncoding.EncodeToString([]byte(time.Time{}.Format(time.RFC3339)))

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, archived, limit, offset).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"cursor\":\""+cursor+"\",\"decks\":[]}", w.Body.String())
	})

	t.Run("正常系_#04", func(t *testing.T) {
		deck := entity.Deck{
			UserId: uid,
		}

		decks := []*entity.Deck{
			&deck,
		}

		archived := false
		limit := 10
		offset := 0
		cursor, err := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archived, limit, cursor).Return(decks, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/decks?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(decks), len(res.Decks))
		require.Equal(t, uid, res.Decks[0].Data.UserId)
		require.Equal(t, base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))), res.Cursor)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		mockUsecase.EXPECT().FindByUserId(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/decks", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		cursor, err := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindByUserIdOnCursor(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/decks?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Create(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         "",
			Name:           "ロスギラ",
			Code:           "",
			PrivateCodeFlg: false,
		}

		param := usecase.NewDeckParam(
			"",
			"ロスギラ",
			"",
			false,
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(deck, nil)

		data := dto.DeckCreateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           "ロスギラ",
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/decks", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         uid,
			Name:           "ロスギラ",
			Code:           "",
			PrivateCodeFlg: false,
		}

		param := usecase.NewDeckParam(
			uid,
			"ロスギラ",
			"",
			false,
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(deck, nil)

		data := dto.DeckCreateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           "ロスギラ",
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/decks", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, errors.New(""))

		data := dto.DeckCreateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           "ロスギラ",
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/decks", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Update(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         "",
			Name:           "ロスギラ",
			Code:           "",
			PrivateCodeFlg: false,
		}

		param := usecase.NewDeckParam(
			"",
			"ロスギラ",
			"",
			false,
		)

		mockUsecase.EXPECT().Update(context.Background(), id, param).Return(deck, nil)

		data := dto.DeckUpdateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           "ロスギラ",
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/decks/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         uid,
			Name:           "ロスギラ",
			Code:           "",
			PrivateCodeFlg: false,
		}

		param := usecase.NewDeckParam(
			uid,
			"ロスギラ",
			"",
			false,
		)

		mockUsecase.EXPECT().Update(context.Background(), id, param).Return(deck, nil)

		data := dto.DeckUpdateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           "ロスギラ",
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/decks/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Update(context.Background(), id, gomock.Any()).Return(nil, errors.New(""))

		data := dto.DeckUpdateRequest{
			DeckRequest: dto.DeckRequest{
				Name:           "ロスギラ",
				Code:           "",
				PrivateCodeFlg: false,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/decks/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Archive(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)
		archivedAt := time.Now().UTC().Truncate(0)

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     archivedAt,
			UserId:         uid,
			Name:           "ロスギラ",
			Code:           "",
			PrivateCodeFlg: false,
		}

		mockUsecase.EXPECT().Archive(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", "/decks/"+id+"/archive", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckArchiveResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Archive(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", "/decks/"+id+"/archive", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Unarchive(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().UTC().Truncate(0)

		deck := &entity.Deck{
			ID:             id,
			CreatedAt:      createdAt,
			ArchivedAt:     time.Time{},
			UserId:         uid,
			Name:           "ロスギラ",
			Code:           "",
			PrivateCodeFlg: false,
		}

		mockUsecase.EXPECT().Unarchive(context.Background(), id).Return(deck, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", "/decks/"+id+"/unarchive", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.DeckUnarchiveResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Unarchive(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PATCH", "/decks/"+id+"/unarchive", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_DeckController_Delete(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestDeckController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/decks/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/decks/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/decks/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
