package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

// stubUnofficialEventUsecase は自由形式イベントユースケースのスタブ。
// mock_usecaseにUnofficialEvent用のモックが存在しないため手書きする。
type stubUnofficialEventUsecase struct {
	event      *entity.UnofficialEvent
	findErr    error
	createErr  error
	gotParam   *usecase.UnofficialEventParam
	createdVal *entity.UnofficialEvent
}

func (s *stubUnofficialEventUsecase) FindById(ctx context.Context, id string) (*entity.UnofficialEvent, error) {
	return s.event, s.findErr
}

func (s *stubUnofficialEventUsecase) Create(ctx context.Context, param *usecase.UnofficialEventParam) (*entity.UnofficialEvent, error) {
	s.gotParam = param
	return s.createdVal, s.createErr
}

func setup4TestUnofficialEventController(t *testing.T, u *stubUnofficialEventUsecase) (*UnofficialEvent, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	r := gin.Default()
	c := NewUnofficialEvent(r, u)
	c.RegisterRoute("")

	return c, secretKey
}

func TestUnofficialEventController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)

	t.Run("GetById", func(t *testing.T) {
		t.Run("正常系_指定IDの自由形式イベントを返す", func(t *testing.T) {
			event := entity.NewUnofficialEvent(id, uid, "自主大会", date)
			c, _ := setup4TestUnofficialEventController(t, &stubUnofficialEventUsecase{event: event})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UnofficialEventsPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			var res dto.UnofficialEventGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, "自主大会", res.Title)
		})

		t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
			c, _ := setup4TestUnofficialEventController(t, &stubUnofficialEventUsecase{findErr: apperror.ErrRecordNotFound})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UnofficialEventsPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, _ := setup4TestUnofficialEventController(t, &stubUnofficialEventUsecase{findErr: errors.New("")})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UnofficialEventsPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("Create", func(t *testing.T) {
		newRequestBody := func(t *testing.T) string {
			t.Helper()
			b, err := json.Marshal(dto.UnofficialEventCreateRequest{
				UnofficialEventRequest: dto.UnofficialEventRequest{Title: "自主大会", Date: date},
			})
			require.NoError(t, err)
			return string(b)
		}

		t.Run("正常系_認証済みユーザのIDでイベントを作成する", func(t *testing.T) {
			stub := &stubUnofficialEventUsecase{createdVal: entity.NewUnofficialEvent(id, uid, "自主大会", date)}
			c, secretKey := setup4TestUnofficialEventController(t, stub)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", UnofficialEventsPath, strings.NewReader(newRequestBody(t)))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			var res dto.UnofficialEventCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusCreated, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, uid, res.UserId)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _ := setup4TestUnofficialEventController(t, &stubUnofficialEventUsecase{})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", UnofficialEventsPath, strings.NewReader(newRequestBody(t)))
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_イベント名が空なら400を返す", func(t *testing.T) {
			c, secretKey := setup4TestUnofficialEventController(t, &stubUnofficialEventUsecase{})

			b, err := json.Marshal(dto.UnofficialEventCreateRequest{
				UnofficialEventRequest: dto.UnofficialEventRequest{Title: "", Date: date},
			})
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", UnofficialEventsPath, strings.NewReader(string(b)))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, secretKey := setup4TestUnofficialEventController(t, &stubUnofficialEventUsecase{createErr: errors.New("")})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", UnofficialEventsPath, strings.NewReader(newRequestBody(t)))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
