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
		"Get":         test_RecordController_Get,
		"GetById":     test_RecordController_GetById,
		"GetByUserId": test_RecordController_GetByUserId,
		"Create":      test_RecordController_Create,
		"Update":      test_RecordController_Update,
		"Delete":      test_RecordController_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_RecordController_Get(t *testing.T) {
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

		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
	})

	t.Run("正常系_#02", func(t *testing.T) {
		record := entity.Record{}
		records := []*entity.Record{
			&record,
		}

		limit := 10
		offset := 0

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", RecordsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
	})

	t.Run("正常系_#03", func(t *testing.T) {
		records := []*entity.Record{}

		limit := 10
		offset := 0
		cursor := base64.StdEncoding.EncodeToString([]byte(time.Time{}.Format(time.RFC3339)))

		mockUsecase.EXPECT().Find(context.Background(), limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", RecordsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"cursor\":\""+cursor+"\",\"records\":[]}", w.Body.String())
	})

	t.Run("正常系_#04", func(t *testing.T) {
		record := entity.Record{}
		records := []*entity.Record{
			&record,
		}

		limit := 10
		offset := 0
		cursor, err := time.Parse(time.RFC3339, time.Now().Local().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))), res.Cursor)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		mockUsecase.EXPECT().Find(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", RecordsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		cursor, err := time.Parse(time.RFC3339, time.Now().Local().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindOnCursor(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_RecordController_GetById(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestRecordController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Local()
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

		req, err := http.NewRequest("GET", RecordsPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, officialEventId, res.OfficialEventId)
		require.Equal(t, privateFlg, res.PrivateFlg)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", RecordsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", RecordsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_RecordController_GetByUserId(t *testing.T) {
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

		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?limit=%d&offset=%d", limit, offset), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, uid, res.Records[0].Data.UserId)
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

		req, err := http.NewRequest("GET", RecordsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, uid, res.Records[0].Data.UserId)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		records := []*entity.Record{}

		limit := 10
		offset := 0
		cursor := base64.StdEncoding.EncodeToString([]byte(time.Time{}.Format(time.RFC3339)))

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", RecordsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"cursor\":\""+cursor+"\",\"records\":[]}", w.Body.String())
	})

	t.Run("正常系_#04", func(t *testing.T) {
		record := entity.Record{
			UserId: uid,
		}

		records := []*entity.Record{
			&record,
		}

		limit := 10
		offset := 0
		cursor, err := time.Parse(time.RFC3339, time.Now().Local().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindByUserIdOnCursor(context.Background(), uid, limit, cursor).Return(records, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordGetByUserIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, limit, res.Limit)
		require.Equal(t, offset, res.Offset)
		require.Equal(t, len(records), len(res.Records))
		require.Equal(t, uid, res.Records[0].Data.UserId)
		require.Equal(t, base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))), res.Cursor)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		limit := 10
		offset := 0

		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", RecordsPath, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		cursor, err := time.Parse(time.RFC3339, time.Now().Local().Format(time.RFC3339))
		require.NoError(t, err)

		mockUsecase.EXPECT().FindByUserIdOnCursor(context.Background(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?cursor=%s", base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339)))), nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_RecordController_Create(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Local()
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

		req, err := http.NewRequest("POST", RecordsPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
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

		createdAt := time.Now().Local()
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

		req, err := http.NewRequest("POST", RecordsPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
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

		req, err := http.NewRequest("POST", RecordsPath, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_RecordController_Update(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Local()
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

		req, err := http.NewRequest("PUT", RecordsPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
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

		createdAt := time.Now().Local()
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

		req, err := http.NewRequest("PUT", RecordsPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, officialEventId, res.OfficialEventId)
		require.Equal(t, privateFlg, res.PrivateFlg)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		officialEventId := uint(10000)
		privateFlg := false

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

		req, err := http.NewRequest("PUT", RecordsPath+"/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.RecordCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

}

func test_RecordController_Delete(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestRecordController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", RecordsPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", RecordsPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", RecordsPath+"/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
