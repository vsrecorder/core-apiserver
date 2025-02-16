package controller

import (
	"context"
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

func setupMock4TestRecordController(t *testing.T) (*mock_repository.MockRecordInterface, *mock_usecase.MockRecordInterface) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	mockUsecase := mock_usecase.NewMockRecordInterface(mockCtrl)

	return mockRepository, mockUsecase
}

func setup4TestRecordController(t *testing.T, r *gin.Engine) (
	*Record,
	*mock_usecase.MockRecordInterface,
) {
	authDisable := true
	mockRepository, mockUsecase := setupMock4TestRecordController(t)

	c := NewRecord(r, mockRepository, mockUsecase)
	c.RegisterRoute("", authDisable)

	return c, mockUsecase
}

func TestRecordController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"Get":         test_Get,
		"GetById":     test_GetById,
		"GetByUserId": test_GetByUserId,
		"Create":      test_Create,
		"Update":      test_Update,
		"Delete":      test_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_Get(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestRecordController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		record := entity.Record{}
		records := []*entity.Record{
			&record,
		}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/records?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
	})

	t.Run("正常系_02", func(t *testing.T) {
		record := entity.Record{}
		records := []*entity.Record{
			&record,
		}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
	})

	t.Run("正常系_03", func(t *testing.T) {
		records := []*entity.Record{}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[]}", w.Body.String())
	})

	t.Run("異常系_#01", func(t *testing.T) {
		mockUsecase.EXPECT().Find(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_GetById(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestRecordController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Truncate(0)
		officialEventId := uint(10000)
		privateFlg := false

		record := &entity.Record{
			ID:              id,
			CreatedAt:       createdAt,
			OfficialEventId: officialEventId,
			PrivateFlg:      privateFlg,
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(record, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, officialEventId, res.OfficialEventId)
		require.Equal(t, privateFlg, res.PrivateFlg)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})
}

func test_GetByUserId(t *testing.T) {
	r := gin.Default()

	// 認証済みとするためにuidをセット
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	r.Use(func(ctx *gin.Context) {
		helper.SetUID(ctx, uid)
	})

	c, mockUsecase := setup4TestRecordController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		record := entity.Record{
			UserId: uid,
		}

		records := []*entity.Record{
			&record,
		}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf("/records?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, uid, res.Records[0].UserId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		record := entity.Record{
			UserId: uid,
		}

		records := []*entity.Record{
			&record,
		}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, uid, res.Records[0].UserId)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		records := []*entity.Record{}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[]}", w.Body.String())
	})

	t.Run("異常系_#01", func(t *testing.T) {
		limit := 10
		offset := 0

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_Create(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Truncate(0)
		officialEventId := uint(10000)
		privateFlg := false

		record := &entity.Record{
			ID:              id,
			CreatedAt:       createdAt,
			OfficialEventId: officialEventId,
			PrivateFlg:      privateFlg,
		}

		param := usecase.NewRecordParam(
			officialEventId,
			"",
			"",
			"",
			"",
			privateFlg,
			"",
			"",
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(record, nil)

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/records", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, officialEventId, res.OfficialEventId)
		require.Equal(t, privateFlg, res.PrivateFlg)
		require.Equal(t, "", res.UserId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Truncate(0)
		officialEventId := uint(10000)
		privateFlg := false

		record := &entity.Record{
			ID:              id,
			CreatedAt:       createdAt,
			OfficialEventId: officialEventId,
			UserId:          uid,
			PrivateFlg:      privateFlg,
		}

		param := usecase.NewRecordParam(
			officialEventId,
			"",
			"",
			uid,
			"",
			privateFlg,
			"",
			"",
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(record, nil)

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/records", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, officialEventId, res.OfficialEventId)
		require.Equal(t, privateFlg, res.PrivateFlg)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestRecordController(t, r)

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, errors.New(""))

		officialEventId := uint(10000)
		privateFlg := false

		data := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: officialEventId,
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      privateFlg,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/records", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_Update(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Truncate(0)

		{
			officialEventId := uint(10000)
			privateFlg := false

			record := &entity.Record{
				ID:              id,
				CreatedAt:       createdAt,
				OfficialEventId: officialEventId,
				PrivateFlg:      privateFlg,
			}

			param := usecase.NewRecordParam(
				officialEventId,
				"",
				"",
				"",
				"",
				privateFlg,
				"",
				"",
			)

			mockUsecase.EXPECT().Create(context.Background(), param).Return(record, nil)

			data := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: officialEventId,
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      privateFlg,
					TCGMeisterURL:   "",
					Memo:            "",
				},
			}

			dataBytes, err := json.Marshal(data)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			req, err := http.NewRequest("POST", "/records", strings.NewReader(string(dataBytes)))
			require.NoError(t, err)

			c.router.ServeHTTP(w, req)

			var res dto.RecordCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusCreated, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, createdAt, res.CreatedAt)
			require.Equal(t, officialEventId, res.OfficialEventId)
			require.Equal(t, privateFlg, res.PrivateFlg)
			require.Equal(t, "", res.UserId)
		}

		{
			officialEventId := uint(10001)
			privateFlg := true

			record := &entity.Record{
				ID:              id,
				CreatedAt:       createdAt,
				OfficialEventId: officialEventId,
				PrivateFlg:      privateFlg,
			}

			param := usecase.NewRecordParam(
				officialEventId,
				"",
				"",
				"",
				"",
				privateFlg,
				"",
				"",
			)

			mockUsecase.EXPECT().Update(context.Background(), id, param).Return(record, nil)

			data := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: officialEventId,
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      privateFlg,
					TCGMeisterURL:   "",
					Memo:            "",
				},
			}

			dataBytes, err := json.Marshal(data)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			req, err := http.NewRequest("PUT", "/records/"+id, strings.NewReader(string(dataBytes)))
			require.NoError(t, err)

			c.router.ServeHTTP(w, req)

			var res dto.RecordCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, createdAt, res.CreatedAt)
			require.Equal(t, officialEventId, res.OfficialEventId)
			require.Equal(t, privateFlg, res.PrivateFlg)
			require.Equal(t, "", res.UserId)
		}
	})

	t.Run("正常系_#02", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Truncate(0)

		{
			officialEventId := uint(10000)
			privateFlg := false

			record := &entity.Record{
				ID:              id,
				CreatedAt:       createdAt,
				OfficialEventId: officialEventId,
				UserId:          uid,
				PrivateFlg:      privateFlg,
			}

			param := usecase.NewRecordParam(
				officialEventId,
				"",
				"",
				uid,
				"",
				privateFlg,
				"",
				"",
			)

			mockUsecase.EXPECT().Create(context.Background(), param).Return(record, nil)

			data := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: officialEventId,
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      privateFlg,
					TCGMeisterURL:   "",
					Memo:            "",
				},
			}

			dataBytes, err := json.Marshal(data)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			req, err := http.NewRequest("POST", "/records", strings.NewReader(string(dataBytes)))
			require.NoError(t, err)

			c.router.ServeHTTP(w, req)

			var res dto.RecordCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusCreated, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, createdAt, res.CreatedAt)
			require.Equal(t, officialEventId, res.OfficialEventId)
			require.Equal(t, privateFlg, res.PrivateFlg)
			require.Equal(t, uid, res.UserId)
		}

		{
			officialEventId := uint(10001)
			privateFlg := true

			record := &entity.Record{
				ID:              id,
				CreatedAt:       createdAt,
				OfficialEventId: officialEventId,
				UserId:          uid,
				PrivateFlg:      privateFlg,
			}

			param := usecase.NewRecordParam(
				officialEventId,
				"",
				"",
				uid,
				"",
				privateFlg,
				"",
				"",
			)

			mockUsecase.EXPECT().Update(context.Background(), id, param).Return(record, nil)

			data := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: officialEventId,
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      privateFlg,
					TCGMeisterURL:   "",
					Memo:            "",
				},
			}

			dataBytes, err := json.Marshal(data)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			req, err := http.NewRequest("PUT", "/records/"+id, strings.NewReader(string(dataBytes)))
			require.NoError(t, err)

			c.router.ServeHTTP(w, req)

			var res dto.RecordCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, createdAt, res.CreatedAt)
			require.Equal(t, officialEventId, res.OfficialEventId)
			require.Equal(t, privateFlg, res.PrivateFlg)
			require.Equal(t, uid, res.UserId)
		}
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Truncate(0)

		{
			officialEventId := uint(10000)
			privateFlg := false

			record := &entity.Record{
				ID:              id,
				CreatedAt:       createdAt,
				OfficialEventId: officialEventId,
				PrivateFlg:      privateFlg,
			}

			param := usecase.NewRecordParam(
				officialEventId,
				"",
				"",
				"",
				"",
				privateFlg,
				"",
				"",
			)

			mockUsecase.EXPECT().Create(context.Background(), param).Return(record, nil)

			data := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: officialEventId,
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      privateFlg,
					TCGMeisterURL:   "",
					Memo:            "",
				},
			}

			dataBytes, err := json.Marshal(data)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			req, err := http.NewRequest("POST", "/records", strings.NewReader(string(dataBytes)))
			require.NoError(t, err)

			c.router.ServeHTTP(w, req)

			var res dto.RecordCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusCreated, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, createdAt, res.CreatedAt)
			require.Equal(t, officialEventId, res.OfficialEventId)
			require.Equal(t, privateFlg, res.PrivateFlg)
			require.Equal(t, "", res.UserId)
		}

		{
			officialEventId := uint(10001)
			privateFlg := true

			param := usecase.NewRecordParam(
				officialEventId,
				"",
				"",
				"",
				"",
				privateFlg,
				"",
				"",
			)

			mockUsecase.EXPECT().Update(context.Background(), id, param).Return(nil, errors.New(""))

			data := dto.RecordCreateRequest{
				RecordRequest: dto.RecordRequest{
					OfficialEventId: officialEventId,
					TonamelEventId:  "",
					FriendId:        "",
					DeckId:          "",
					PrivateFlg:      privateFlg,
					TCGMeisterURL:   "",
					Memo:            "",
				},
			}

			dataBytes, err := json.Marshal(data)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			req, err := http.NewRequest("PUT", "/records/"+id, strings.NewReader(string(dataBytes)))
			require.NoError(t, err)

			c.router.ServeHTTP(w, req)

			{
				var res dto.RecordCreateResponse
				err := json.Unmarshal(w.Body.Bytes(), &res)
				require.NoError(t, err)

				require.Equal(t, http.StatusInternalServerError, w.Code)
			}
		}
	})

}

func test_Delete(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestRecordController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/records/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/records/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/records/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
