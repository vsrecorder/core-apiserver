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

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

// stubUserPlayerUsecase はプレイヤーID連携ユースケースのスタブ。
// mock_usecaseにUserPlayer用のモックが存在しないため手書きする。
type stubUserPlayerUsecase struct {
	userPlayer *entity.UserPlayer
	findErr    error
	created    *entity.UserPlayer
	createErr  error
}

func (s stubUserPlayerUsecase) FindByUserId(ctx context.Context, userId string) (*entity.UserPlayer, error) {
	return s.userPlayer, s.findErr
}

func (s stubUserPlayerUsecase) Create(ctx context.Context, param *usecase.UserPlayerCreateParam) (*entity.UserPlayer, error) {
	return s.created, s.createErr
}

func setup4TestUserPlayerController(t *testing.T, u stubUserPlayerUsecase, linkingEnabled bool) (*UserPlayer, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := gin.Default()
	c := NewUserPlayer(logger, r, u, linkingEnabled)
	c.RegisterRoute("")

	return c, secretKey
}

func TestUserPlayerController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	playerId := "1234567890123456"

	newUserPlayer := func() *entity.UserPlayer {
		return entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN2", time.Now().Local(), uid, playerId)
	}

	t.Run("GetByUID", func(t *testing.T) {
		t.Run("正常系_紐付けを返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{
				userPlayer: newUserPlayer(),
			}, true)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UserPlayersPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			var res dto.UserPlayerGetResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, playerId, res.PlayerId)
		})

		// チャンピオンシップポイント関連のフィールドは返さない
		t.Run("正常系_レスポンスにランキング関連のフィールドを含めない", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{
				userPlayer: newUserPlayer(),
			}, true)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UserPlayersPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.NotContains(t, w.Body.String(), "champion_ship_point")
			require.NotContains(t, w.Body.String(), "ranking_date")
		})

		t.Run("異常系_紐付けが無ければ404を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{findErr: apperror.ErrRecordNotFound}, true)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UserPlayersPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _ := setup4TestUserPlayerController(t, stubUserPlayerUsecase{}, true)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UserPlayersPath, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_連携機能が無効なら503を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{}, false)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UserPlayersPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusServiceUnavailable, w.Code)
		})
	})

	t.Run("Create", func(t *testing.T) {
		newCreateRequest := func(t *testing.T, playerId string) *http.Request {
			t.Helper()
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: playerId})
			require.NoError(t, err)
			req, err := http.NewRequest("POST", UserPlayersPath, strings.NewReader(string(b)))
			require.NoError(t, err)
			return req
		}

		t.Run("正常系_紐付けを作成して201を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{created: newUserPlayer()}, true)

			w := httptest.NewRecorder()
			req := newCreateRequest(t, "6000000000000001")
			setJWTAuthHeader(t, req, "ctrl-create-ok-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusCreated, w.Code)
		})

		t.Run("異常系_紐付けから1ヶ月未満の変更は409を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{createErr: apperror.ErrLocked}, true)

			w := httptest.NewRecorder()
			req := newCreateRequest(t, "6000000000000002")
			setJWTAuthHeader(t, req, "ctrl-create-locked-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusConflict, w.Code)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _ := setup4TestUserPlayerController(t, stubUserPlayerUsecase{}, true)

			w := httptest.NewRecorder()
			req := newCreateRequest(t, "6000000000000003")
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_連携機能が無効なら503を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{}, false)

			w := httptest.NewRecorder()
			req := newCreateRequest(t, "6000000000000004")
			setJWTAuthHeader(t, req, "ctrl-create-disabled-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusServiceUnavailable, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{createErr: errors.New("")}, true)

			w := httptest.NewRecorder()
			req := newCreateRequest(t, "6000000000000006")
			setJWTAuthHeader(t, req, "ctrl-create-error-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})
}
