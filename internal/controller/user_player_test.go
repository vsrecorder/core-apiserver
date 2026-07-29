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
	userPlayer   *entity.UserPlayer
	findErr      error
	ranking      *entity.PlayerRanking
	rankingErr   error
	created      *entity.UserPlayer
	createErr    error
	avatar       *entity.PokemonAvatar
	challengeErr error
}

func (s stubUserPlayerUsecase) FindByUserId(ctx context.Context, userId string) (*entity.UserPlayer, error) {
	return s.userPlayer, s.findErr
}

func (s stubUserPlayerUsecase) FindLatestPlayerRanking(ctx context.Context, playerId string) (*entity.PlayerRanking, error) {
	return s.ranking, s.rankingErr
}

func (s stubUserPlayerUsecase) Create(ctx context.Context, param *usecase.UserPlayerCreateParam) (*entity.UserPlayer, error) {
	return s.created, s.createErr
}

func (s stubUserPlayerUsecase) IssueChallengeAvatar(ctx context.Context, currentAvatarImage string) (*entity.PokemonAvatar, error) {
	return s.avatar, s.challengeErr
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
		t.Run("正常系_紐付けと最新ランキングを返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{
				userPlayer: newUserPlayer(),
				ranking:    &entity.PlayerRanking{PlayerId: playerId, ChampionShipPoint: 250},
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

		t.Run("正常系_ランキング履歴なしでも紐付けは返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{
				userPlayer: newUserPlayer(),
				rankingErr: apperror.ErrRecordNotFound,
			}, true)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UserPlayersPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
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

	t.Run("Challenge", func(t *testing.T) {
		newChallengeRequest := func(t *testing.T) *http.Request {
			t.Helper()

			b, err := json.Marshal(dto.UserPlayerChallengeRequest{
				CurrentAvatarImage: "https://example.com/current-avatar.png",
			})
			require.NoError(t, err)

			req, err := http.NewRequest("POST", UserPlayersPath+"/challenge", strings.NewReader(string(b)))
			require.NoError(t, err)

			return req
		}

		t.Run("正常系_チャレンジ用アバターを返す", func(t *testing.T) {
			avatar := entity.NewPokemonAvatar(1, "ピカチュウ", "https://example.com/other-avatar.png", "詳細")
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{avatar: avatar}, true)

			w := httptest.NewRecorder()
			req := newChallengeRequest(t)
			setJWTAuthHeader(t, req, "ctrl-challenge-ok-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var res dto.UserPlayerChallengeResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			require.Equal(t, "https://example.com/other-avatar.png", res.AvatarImageURL)
			require.Equal(t, "ピカチュウ", res.AvatarTitle)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{challengeErr: errors.New("")}, true)

			w := httptest.NewRecorder()
			req := newChallengeRequest(t)
			setJWTAuthHeader(t, req, "ctrl-challenge-error-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _ := setup4TestUserPlayerController(t, stubUserPlayerUsecase{}, true)

			w := httptest.NewRecorder()
			req := newChallengeRequest(t)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_連携機能が無効なら503を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{}, false)

			w := httptest.NewRecorder()
			req := newChallengeRequest(t)
			setJWTAuthHeader(t, req, "ctrl-challenge-disabled-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusServiceUnavailable, w.Code)
		})
	})

	t.Run("Create", func(t *testing.T) {
		newCreateRequest := func(t *testing.T, playerId string) *http.Request {
			t.Helper()
			b, err := json.Marshal(dto.UserPlayerCreateRequest{PlayerId: playerId, VerificationToken: "token"})
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

		t.Run("異常系_別ユーザに紐付け済みのプレイヤーIDは409を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{createErr: apperror.ErrAlreadyExists}, true)

			w := httptest.NewRecorder()
			req := newCreateRequest(t, "6000000000000003")
			setJWTAuthHeader(t, req, "ctrl-create-linked-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusConflict, w.Code)
		})

		t.Run("異常系_検証済みトークンが無効なら400を返す", func(t *testing.T) {
			c, secretKey := setup4TestUserPlayerController(t, stubUserPlayerUsecase{createErr: apperror.ErrInvalidVerification}, true)

			w := httptest.NewRecorder()
			req := newCreateRequest(t, "6000000000000004")
			setJWTAuthHeader(t, req, "ctrl-create-invalid-user", secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
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
