package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
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
	mockUsecase := setupMock(t)
	c := NewRecord(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestRecordController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	c, mockUsecase := setup(t, r)

	for scenario, fn := range map[string]func(
		t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface,
	){
		"Get":    test_Get,
		"Create": test_Create,
		"Update": test_Update,
		"Delete": test_Delete,
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
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[{\"ID\":\"\",\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"OfficialEventId\":0,\"TonamelEventId\":\"\",\"FriendId\":\"\",\"UserId\":\"\",\"DeckId\":\"\",\"PrivateFlg\":false,\"TCGMeisterURL\":\"\",\"Memo\":\"\"}]}", w.Body.String())
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
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[{\"ID\":\"\",\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"OfficialEventId\":0,\"TonamelEventId\":\"\",\"FriendId\":\"\",\"UserId\":\"\",\"DeckId\":\"\",\"PrivateFlg\":false,\"TCGMeisterURL\":\"\",\"Memo\":\"\"}]}", w.Body.String())
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
		require.Equal(t, "{\"limit\":10,\"offset\":0,\"records\":[{\"ID\":\"\",\"CreatedAt\":\"0001-01-01T00:00:00Z\",\"OfficialEventId\":0,\"TonamelEventId\":\"\",\"FriendId\":\"\",\"UserId\":\"\",\"DeckId\":\"\",\"PrivateFlg\":false,\"TCGMeisterURL\":\"\",\"Memo\":\"\"}]}", w.Body.String())
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

func test_Create(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)

	rcr := dto.RecordCreateRequest{
		OfficialEventId: 10000,
		TonamelEventId:  "",
		FriendId:        "",
		DeckId:          "",
		PrivateFlg:      false,
		TCGMeisterURL:   "",
		Memo:            "",
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

		require.Equal(t, reflect.TypeOf(""), reflect.TypeOf(res.ID))
		require.Equal(t, reflect.TypeOf(time.Time{}), reflect.TypeOf(res.CreatedAt))
		require.Equal(t, rcr.OfficialEventId, res.OfficialEventId)
		require.Equal(t, rcr.PrivateFlg, res.PrivateFlg)
	}
}

func test_Update(t *testing.T, c *Record, mockUsecase *mock_usecase.MockRecordInterface) {
	mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)

	rcr := dto.RecordCreateRequest{
		OfficialEventId: 10000,
		TonamelEventId:  "",
		FriendId:        "",
		DeckId:          "",
		PrivateFlg:      false,
		TCGMeisterURL:   "",
		Memo:            "",
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

		require.Equal(t, rcr.OfficialEventId, res.OfficialEventId)
		require.Equal(t, rcr.PrivateFlg, res.PrivateFlg)

		id := res.ID

		mockUsecase.EXPECT().Update(context.Background(), id, gomock.Any()).Return(nil)

		rur := dto.RecordCreateRequest{
			OfficialEventId: 10001,
			TonamelEventId:  "",
			FriendId:        "",
			DeckId:          "",
			PrivateFlg:      true,
			TCGMeisterURL:   "",
			Memo:            "",
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
