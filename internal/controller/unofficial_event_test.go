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
	updateErr  error
	deleteErr  error
	gotParam   *usecase.UnofficialEventParam
	createdVal *entity.UnofficialEvent
	updatedVal *entity.UnofficialEvent
	deletedId  string
}

func (s *stubUnofficialEventUsecase) FindById(ctx context.Context, id string) (*entity.UnofficialEvent, error) {
	return s.event, s.findErr
}

func (s *stubUnofficialEventUsecase) Create(ctx context.Context, param *usecase.UnofficialEventParam) (*entity.UnofficialEvent, error) {
	s.gotParam = param
	return s.createdVal, s.createErr
}

func (s *stubUnofficialEventUsecase) Update(ctx context.Context, id string, param *usecase.UnofficialEventParam) (*entity.UnofficialEvent, error) {
	s.gotParam = param
	return s.updatedVal, s.updateErr
}

func (s *stubUnofficialEventUsecase) Delete(ctx context.Context, id string) error {
	s.deletedId = id
	return s.deleteErr
}

// stubUnofficialEventRepository は所有者チェック(authorization)に使うリポジトリのスタブ。
type stubUnofficialEventRepository struct {
	event   *entity.UnofficialEvent
	findErr error
}

func (s *stubUnofficialEventRepository) FindById(ctx context.Context, id string) (*entity.UnofficialEvent, error) {
	return s.event, s.findErr
}

func (s *stubUnofficialEventRepository) Save(ctx context.Context, e *entity.UnofficialEvent) error {
	return nil
}

func (s *stubUnofficialEventRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func setup4TestUnofficialEventController(t *testing.T, u *stubUnofficialEventUsecase) (*UnofficialEvent, string) {
	t.Helper()

	return setup4TestUnofficialEventControllerWithRepository(t, u, &stubUnofficialEventRepository{})
}

func setup4TestUnofficialEventControllerWithRepository(
	t *testing.T,
	u *stubUnofficialEventUsecase,
	repo *stubUnofficialEventRepository,
) (*UnofficialEvent, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	r := gin.Default()
	c := NewUnofficialEvent(r, repo, u)
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

	t.Run("Update", func(t *testing.T) {
		newRequestBody := func(t *testing.T, title string) string {
			t.Helper()
			b, err := json.Marshal(dto.UnofficialEventUpdateRequest{
				UnofficialEventRequest: dto.UnofficialEventRequest{Title: title, Date: date},
			})
			require.NoError(t, err)
			return string(b)
		}

		// 所有者として認可を通すリポジトリスタブ
		ownedRepository := func() *stubUnofficialEventRepository {
			return &stubUnofficialEventRepository{event: entity.NewUnofficialEvent(id, uid, "自主大会", date)}
		}

		t.Run("正常系_イベント名と開催日を更新する", func(t *testing.T) {
			stub := &stubUnofficialEventUsecase{updatedVal: entity.NewUnofficialEvent(id, uid, "身内対戦会", date)}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, stub, ownedRepository())

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", UnofficialEventsPath+"/"+id, strings.NewReader(newRequestBody(t, "身内対戦会")))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			var res dto.UnofficialEventUpdateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, id, res.ID)
			require.Equal(t, uid, res.UserId)
			require.Equal(t, "身内対戦会", res.Title)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _ := setup4TestUnofficialEventControllerWithRepository(t, &stubUnofficialEventUsecase{}, ownedRepository())

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", UnofficialEventsPath+"/"+id, strings.NewReader(newRequestBody(t, "身内対戦会")))
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_他人のイベントなら403を返す", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{
				event: entity.NewUnofficialEvent(id, "KBp7roRDZobZg1t0OPzFR1kvLeO2", "自主大会", date),
			}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, &stubUnofficialEventUsecase{}, repo)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", UnofficialEventsPath+"/"+id, strings.NewReader(newRequestBody(t, "身内対戦会")))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
		})

		t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{findErr: apperror.ErrRecordNotFound}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, &stubUnofficialEventUsecase{}, repo)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", UnofficialEventsPath+"/"+id, strings.NewReader(newRequestBody(t, "身内対戦会")))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_イベント名が空なら400を返す", func(t *testing.T) {
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, &stubUnofficialEventUsecase{}, ownedRepository())

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", UnofficialEventsPath+"/"+id, strings.NewReader(newRequestBody(t, "")))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			stub := &stubUnofficialEventUsecase{updateErr: errors.New("")}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, stub, ownedRepository())

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", UnofficialEventsPath+"/"+id, strings.NewReader(newRequestBody(t, "身内対戦会")))
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		// 所有者として認可を通すリポジトリスタブ
		ownedRepository := func() *stubUnofficialEventRepository {
			return &stubUnofficialEventRepository{event: entity.NewUnofficialEvent(id, uid, "自主大会", date)}
		}

		t.Run("正常系_指定IDのイベントを削除する", func(t *testing.T) {
			stub := &stubUnofficialEventUsecase{}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, stub, ownedRepository())

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", UnofficialEventsPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNoContent, w.Code)
			require.Equal(t, id, stub.deletedId)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _ := setup4TestUnofficialEventControllerWithRepository(t, &stubUnofficialEventUsecase{}, ownedRepository())

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", UnofficialEventsPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_他人のイベントなら403を返す", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{
				event: entity.NewUnofficialEvent(id, "KBp7roRDZobZg1t0OPzFR1kvLeO2", "自主大会", date),
			}
			stub := &stubUnofficialEventUsecase{}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, stub, repo)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", UnofficialEventsPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
			require.Empty(t, stub.deletedId)
		})

		t.Run("異常系_存在しないIDは404を返す", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{findErr: apperror.ErrRecordNotFound}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, &stubUnofficialEventUsecase{}, repo)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", UnofficialEventsPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			stub := &stubUnofficialEventUsecase{deleteErr: errors.New("")}
			c, secretKey := setup4TestUnofficialEventControllerWithRepository(t, stub, ownedRepository())

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", UnofficialEventsPath+"/"+id, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
