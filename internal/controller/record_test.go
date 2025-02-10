package controller

import (
	"context"
	"encoding/json"
	"errors"
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
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func setupMock(t *testing.T) *mock_usecase.MockRecordInterface {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockRecordInterface(mockCtrl)

	return mockUsecase
}

func setup(t *testing.T, r *gin.Engine) (
	*Record,
	*mock_usecase.MockRecordInterface,
) {
	authDisable := true
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	mockUsecase := setupMock(t)

	c := NewRecord(r, mockRepository, mockUsecase)
	c.RegisterRoute("", authDisable)

	return c, mockUsecase
}

func TestRecordController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	c, mockUsecase := setup(t, r)

	for scenario, fn := range map[string]func(
		t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface,
	){
		"Get":         test_Get,
		"GetById":     test_GetById,
		"GetByUserId": test_GetByUserId,
		"Create":      test_Create,
		"Update":      test_Update,
		"Delete":      test_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, c, mockUsecase)
		})
	}
}

func test_Get(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	{
		record := entity.Record{}
		records := []*entity.Record{
			&record,
		}
		mockUsecase.EXPECT().Find(context.Background(), 10, 0).Return(records, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records?limit=10&offset=0", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[{\"id\":\"\",\"created_at\":\"0001-01-01T00:00:00Z\",\"official_event_id\":0,\"tonamel_event_id\":\"\",\"friend_id\":\"\",\"user_id\":\"\",\"deck_id\":\"\",\"private_flg\":false,\"tcg_meister_url\":\"\",\"memo\":\"\"}]}", w.Body.String())
	}

	{
		record := entity.Record{}
		records := []*entity.Record{
			&record,
		}
		mockUsecase.EXPECT().Find(context.Background(), 10, 0).Return(records, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[{\"id\":\"\",\"created_at\":\"0001-01-01T00:00:00Z\",\"official_event_id\":0,\"tonamel_event_id\":\"\",\"friend_id\":\"\",\"user_id\":\"\",\"deck_id\":\"\",\"private_flg\":false,\"tcg_meister_url\":\"\",\"memo\":\"\"}]}", w.Body.String())
	}

	{
		record := entity.Record{}
		records := []*entity.Record{
			&record,
		}
		mockUsecase.EXPECT().Find(context.Background(), 10, 0).Return(records, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records?limit=0&offset=0", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[{\"id\":\"\",\"created_at\":\"0001-01-01T00:00:00Z\",\"official_event_id\":0,\"tonamel_event_id\":\"\",\"friend_id\":\"\",\"user_id\":\"\",\"deck_id\":\"\",\"private_flg\":false,\"tcg_meister_url\":\"\",\"memo\":\"\"}]}", w.Body.String())
	}

	{

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records?limit=a&offset=0", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, "{\"message\":\"bad request\"}", w.Body.String())
	}

	{

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records?limit=10&offset=a", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, "{\"message\":\"bad request\"}", w.Body.String())
	}
}

func test_GetById(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	id, err := generateId()
	require.NoError(t, err)

	{
		record := &entity.Record{
			ID:              id,
			OfficialEventId: 10000,
			PrivateFlg:      false,
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(record, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "{\"id\":\""+id+"\",\"created_at\":\"0001-01-01T00:00:00Z\",\"official_event_id\":10000,\"tonamel_event_id\":\"\",\"friend_id\":\"\",\"user_id\":\"\",\"deck_id\":\"\",\"private_flg\":false,\"tcg_meister_url\":\"\",\"memo\":\"\"}", w.Body.String())
	}

	{
		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	}
}

func test_GetByUserId(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	{
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		record := entity.Record{
			UserId: uid,
		}

		records := []*entity.Record{
			&record,
		}

		r := gin.Default()

		// 認証済みとするためにuidをセット
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup(t, r)
		mockUsecase.EXPECT().FindByUserId(context.Background(), uid, 10, 0).Return(records, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/records?limit=10&offset=0", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[{\"id\":\"\",\"created_at\":\"0001-01-01T00:00:00Z\",\"official_event_id\":0,\"tonamel_event_id\":\"\",\"friend_id\":\"\",\"user_id\":\""+uid+"\",\"deck_id\":\"\",\"private_flg\":false,\"tcg_meister_url\":\"\",\"memo\":\"\"}]}", w.Body.String())
	}
}

func test_Create(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	id, err := generateId()
	require.NoError(t, err)

	createdAt := time.Now().Truncate(0)

	record := &entity.Record{
		ID:              id,
		CreatedAt:       createdAt,
		OfficialEventId: 10000,
		PrivateFlg:      false,
	}

	mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(record, nil)

	rcr := dto.RecordCreateRequest{
		RecordRequest: dto.RecordRequest{
			OfficialEventId: 10000,
			TonamelEventId:  "",
			FriendId:        "",
			DeckId:          "",
			PrivateFlg:      false,
			TCGMeisterURL:   "",
			Memo:            "",
		},
	}

	rcrBytes, err := json.Marshal(rcr)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/records", strings.NewReader(string(rcrBytes)))
	c.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	{
		var res dto.RecordCreateResponse
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)

		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, rcr.OfficialEventId, res.OfficialEventId)
		require.Equal(t, rcr.PrivateFlg, res.PrivateFlg)
		require.Equal(t, "", res.UserId)
	}
}

func test_Update(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	id, err := generateId()
	require.NoError(t, err)

	createdAt := time.Now().Truncate(0)

	record := &entity.Record{
		ID:              id,
		CreatedAt:       createdAt,
		OfficialEventId: 10000,
		PrivateFlg:      false,
	}

	mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(record, nil)

	rcr := dto.RecordCreateRequest{
		RecordRequest: dto.RecordRequest{
			OfficialEventId: 10000,
			TonamelEventId:  "",
			FriendId:        "",
			DeckId:          "",
			PrivateFlg:      false,
			TCGMeisterURL:   "",
			Memo:            "",
		},
	}

	rcrBytes, err := json.Marshal(rcr)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/records", strings.NewReader(string(rcrBytes)))
	c.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	{
		var res dto.RecordCreateResponse
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)

		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, rcr.OfficialEventId, res.OfficialEventId)
		require.Equal(t, rcr.PrivateFlg, res.PrivateFlg)
		require.Equal(t, "", res.UserId)

		id := res.ID

		record := &entity.Record{
			ID:              id,
			CreatedAt:       createdAt,
			OfficialEventId: 10001,
			PrivateFlg:      true,
		}

		mockUsecase.EXPECT().Update(context.Background(), id, gomock.Any()).Return(record, nil)

		rur := dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				OfficialEventId: 10001,
				TonamelEventId:  "",
				FriendId:        "",
				DeckId:          "",
				PrivateFlg:      true,
				TCGMeisterURL:   "",
				Memo:            "",
			},
		}

		rurBytes, err := json.Marshal(rur)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/records/"+id, strings.NewReader(string(rurBytes)))
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		{
			var res dto.RecordCreateResponse
			err := json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)

			require.Equal(t, id, res.ID)
			require.Equal(t, createdAt, res.CreatedAt)
			require.Equal(t, rur.OfficialEventId, res.OfficialEventId)
			require.Equal(t, rur.PrivateFlg, res.PrivateFlg)
			require.Equal(t, "", res.UserId)
		}
	}
}

func test_Delete(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	{
		id, err := generateId()
		require.NoError(t, err)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/records/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusAccepted, w.Code)
	}

	{
		id, err := generateId()
		require.NoError(t, err)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/records/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, "{\"message\":\"record not found\"}", w.Body.String())
	}

	{
		id, err := generateId()
		require.NoError(t, err)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/records/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Equal(t, "{\"message\":\"internal server error\"}", w.Body.String())
	}
}
