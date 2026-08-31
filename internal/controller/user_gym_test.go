package controller

import (
	"encoding/json"
	"errors"
	"fmt"
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

const testUserGymUID = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

func setup4TestUserGymController(t *testing.T) (
	*UserGym,
	*mock_usecase.MockUserGymInterface,
	string,
) {
	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockUserGymInterface(mockCtrl)

	r := gin.Default()
	c := NewUserGym(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func newUserGymRequest(t *testing.T, method string, path string, body string, uid string, secretKey string) *http.Request {
	t.Helper()

	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(method, path, nil)
	} else {
		req, err = http.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	require.NoError(t, err)

	if uid != "" {
		setJWTAuthHeader(t, req, uid, secretKey)
	}

	return req
}

func newTestUserGymView(shopId uint, name string) *entity.UserGymView {
	return entity.NewUserGymView(
		entity.NewShop(shopId, name, "194-0013", 13, "東京都", "町田市原町田", "", "", ""),
		time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
	)
}

func TestUserGymController(t *testing.T) {
	myGymsPath := UsersPath + MyGymsPath

	t.Run("Get", func(t *testing.T) {
		t.Run("正常系_登録済みのMyジムを上限つきで返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			views := []*entity.UserGymView{
				newTestUserGymView(10317, "カードショップA"),
				newTestUserGymView(10318, "カードショップB"),
			}

			mockUsecase.EXPECT().Find(gomock.Any(), testUserGymUID).Return(views, nil)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "GET", myGymsPath, "", testUserGymUID, secretKey))

			var res dto.UserGymGetResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, usecase.MaxUserGymsPerUser, res.Limit)
			require.Equal(t, 2, res.Count)
			require.Equal(t, uint(10317), res.UserGyms[0].Shop.ID)
			require.Equal(t, "カードショップA", res.UserGyms[0].Shop.Name)
		})

		t.Run("正常系_未登録なら空配列を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			mockUsecase.EXPECT().Find(gomock.Any(), testUserGymUID).Return([]*entity.UserGymView{}, nil)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "GET", myGymsPath, "", testUserGymUID, secretKey))

			var res dto.UserGymGetResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, 0, res.Count)
			require.NotNil(t, res.UserGyms)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _, secretKey := setup4TestUserGymController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "GET", myGymsPath, "", "", secretKey))

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_登録したMyジムを201で返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			view := newTestUserGymView(10317, "カードショップA")
			mockUsecase.EXPECT().Create(gomock.Any(), testUserGymUID, uint(10317)).Return(view, nil)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "POST", myGymsPath, `{"shop_id":10317}`, testUserGymUID, secretKey))

			var res dto.UserGymCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusCreated, w.Code)
			require.Equal(t, uint(10317), res.Shop.ID)
		})

		t.Run("異常系_上限に達していたら409を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			mockUsecase.EXPECT().Create(gomock.Any(), testUserGymUID, uint(10317)).Return(nil, apperror.ErrTooManyUserGyms)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "POST", myGymsPath, `{"shop_id":10317}`, testUserGymUID, secretKey))

			require.Equal(t, http.StatusConflict, w.Code)
		})

		t.Run("異常系_登録済みの店舗なら409を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			mockUsecase.EXPECT().Create(gomock.Any(), testUserGymUID, uint(10317)).Return(nil, apperror.ErrAlreadyExists)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "POST", myGymsPath, `{"shop_id":10317}`, testUserGymUID, secretKey))

			require.Equal(t, http.StatusConflict, w.Code)
		})

		t.Run("異常系_存在しない店舗なら404を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			mockUsecase.EXPECT().Create(gomock.Any(), testUserGymUID, uint(99999999)).Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "POST", myGymsPath, `{"shop_id":99999999}`, testUserGymUID, secretKey))

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_shop_idが無ければ400を返す", func(t *testing.T) {
			c, _, secretKey := setup4TestUserGymController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "POST", myGymsPath, `{}`, testUserGymUID, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _, secretKey := setup4TestUserGymController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "POST", myGymsPath, `{"shop_id":10317}`, "", secretKey))

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_解除したら204を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			mockUsecase.EXPECT().Delete(gomock.Any(), testUserGymUID, uint(10317)).Return(nil)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "DELETE", myGymsPath+"/10317", "", testUserGymUID, secretKey))

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_shop_idが数値でなければ400を返す", func(t *testing.T) {
			c, _, secretKey := setup4TestUserGymController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "DELETE", myGymsPath+"/abc", "", testUserGymUID, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("GetOfficialEvents", func(t *testing.T) {
		eventsPath := myGymsPath + OfficialEventsPath

		t.Run("正常系_期間内のイベントをMyジムと併せて返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			views := []*entity.UserGymView{newTestUserGymView(10317, "カードショップA")}
			officialEvents := []*entity.OfficialEvent{
				{ID: 606466, ShopId: 10317, ShopName: "カードショップA"},
			}

			startDate := time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local)
			endDate := time.Date(2026, 9, 14, 0, 0, 0, 0, time.Local)

			mockUsecase.EXPECT().FindOfficialEvents(gomock.Any(), testUserGymUID, startDate, endDate).Return(views, officialEvents, nil)

			path := fmt.Sprintf("%s?start_date=2026-08-31&end_date=2026-09-14", eventsPath)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "GET", path, "", testUserGymUID, secretKey))

			var res dto.UserGymOfficialEventGetResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, 1, res.Count)
			require.Equal(t, uint(606466), res.OfficialEvents[0].ID)
			require.Len(t, res.UserGyms, 1)
			require.Equal(t, usecase.MaxUserGymsPerUser, res.Limit)
		})

		t.Run("異常系_期間が逆順なら400を返す", func(t *testing.T) {
			c, _, secretKey := setup4TestUserGymController(t)

			path := fmt.Sprintf("%s?start_date=2026-09-14&end_date=2026-08-31", eventsPath)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "GET", path, "", testUserGymUID, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_期間が長すぎれば400を返す", func(t *testing.T) {
			c, _, secretKey := setup4TestUserGymController(t)

			path := fmt.Sprintf("%s?start_date=2026-01-01&end_date=2026-12-31", eventsPath)
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "GET", path, "", testUserGymUID, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースが失敗したら500を返す", func(t *testing.T) {
			c, mockUsecase, secretKey := setup4TestUserGymController(t)

			mockUsecase.EXPECT().FindOfficialEvents(gomock.Any(), testUserGymUID, gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("error"))

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserGymRequest(t, "GET", eventsPath, "", testUserGymUID, secretKey))

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}

// Myジムのパスは /users 配下の静的セグメントで、/users/:id(ユーザ取得)や
// /users/:id/badges と同じ木に載る。gin は同じ位置に名前の違うワイルドカードが
// 来ると登録時に panic するため、実際に同居させて解決できることを確かめる。
// main.go では両方が同じ Engine に登録されるので、ここが壊れると起動しなくなる。
func TestUserGymRouteCoexistsWithUserRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)

	r := gin.Default()

	require.NotPanics(t, func() {
		NewUser(
			slog.Default(),
			r,
			mock_repository.NewMockUserInterface(mockCtrl),
			mock_usecase.NewMockUserInterface(mockCtrl),
		).RegisterRoute("")

		NewUserGym(r, mock_usecase.NewMockUserGymInterface(mockCtrl)).RegisterRoute("")

		NewBadge(
			r,
			mock_usecase.NewMockBadgeInterface(mockCtrl),
			mock_repository.NewMockChampionshipSeriesInterface(mockCtrl),
		).RegisterRoute("")
	})

	// /users/my_gyms が /users/:id のハンドラ(GetById)に吸われていないことを、
	// 認証の有無で見分ける。GetById は認証不要なので、吸われていれば 200 が返る。
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newUserGymRequest(t, "GET", UsersPath+MyGymsPath, "", "", secretKey))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
