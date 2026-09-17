package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestOpponentDeckCandidateController(t *testing.T) (*OpponentDeckCandidate, *mock_usecase.MockOpponentDeckCandidateInterface, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockOpponentDeckCandidateInterface(mockCtrl)

	r := gin.Default()
	c := NewOpponentDeckCandidate(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestOpponentDeckCandidateController_Get(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	path := MatchesPath + OpponentDeckCandidatesPath

	t.Run("正常系_候補を出現回数順に返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckCandidateController(t)

		candidates := []*entity.OpponentDeckCandidate{
			entity.NewOpponentDeckCandidate("ロストバレット", []*entity.PokemonSprite{
				entity.NewPokemonSpriteWithPosition("0887", 1),
				entity.NewPokemonSpriteWithPosition("0006", 2),
			}, 12),
			entity.NewOpponentDeckCandidate("サーナイトex", nil, 7),
		}
		mockUsecase.EXPECT().FindOpponentDeckCandidates(gomock.Any(), 5).Return(candidates, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+"?limit=5", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		var res dto.OpponentDeckCandidatesGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 5, res.Limit)
		require.Len(t, res.Data, 2)
		require.Equal(t, "ロストバレット", res.Data[0].OpponentsDeckInfo)
		require.Equal(t, 12, res.Data[0].Count)
		require.Len(t, res.Data[0].PokemonSprites, 2)
		require.Equal(t, "0887", res.Data[0].PokemonSprites[0].ID)
		require.Equal(t, uint(1), res.Data[0].PokemonSprites[0].Position)
		require.Equal(t, uint(2), res.Data[0].PokemonSprites[1].Position)
		// スプライト無しは空配列(null にしない)
		require.NotNil(t, res.Data[1].PokemonSprites)
		require.Empty(t, res.Data[1].PokemonSprites)
	})

	t.Run("正常系_limit未指定は既定値で、上限超過は上限に丸める", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckCandidateController(t)

		mockUsecase.EXPECT().FindOpponentDeckCandidates(gomock.Any(), helper.DefaultLimit).Return([]*entity.OpponentDeckCandidate{}, nil)
		mockUsecase.EXPECT().FindOpponentDeckCandidates(gomock.Any(), helper.MaxLimit).Return([]*entity.OpponentDeckCandidate{}, nil)

		for _, query := range []string{"", "?limit=100000"} {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path+query, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("異常系_limitが数値でなければ400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckCandidateController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+"?limit=abc", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestOpponentDeckCandidateController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckCandidateController(t)

		mockUsecase.EXPECT().FindOpponentDeckCandidates(gomock.Any(), helper.DefaultLimit).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
