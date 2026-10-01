package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func setup4TestOpponentDeckController(t *testing.T) (*OpponentDeck, *mock_usecase.MockOpponentDeckInterface, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockOpponentDeckInterface(mockCtrl)

	r := gin.Default()
	c := NewOpponentDeck(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestOpponentDeckController_Get(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	path := MatchesPath + OpponentDecksPath

	t.Run("正常系_自分の相手デッキの組み合わせをスプライト付きで返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		decks := []*entity.OpponentDeck{
			entity.NewOpponentDeck("ドラパルトex", []*entity.PokemonSprite{
				entity.NewPokemonSpriteWithPosition("0887", 1),
				entity.NewPokemonSpriteWithPosition("0006", 2),
			}, 12, "2026-09-28"),
			entity.NewOpponentDeck("ドラパ", nil, 3, "2026-08-01"),
		}
		mockUsecase.EXPECT().FindByUserId(gomock.Any(), uid).Return(decks, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		var res dto.OpponentDecksGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Len(t, res.Data, 2)
		require.Equal(t, "ドラパルトex", res.Data[0].OpponentsDeckInfo)
		require.Equal(t, 12, res.Data[0].Count)
		require.Equal(t, "2026-09-28", res.Data[0].LastEventDate)
		require.Len(t, res.Data[0].PokemonSprites, 2)
		require.Equal(t, uint(2), res.Data[0].PokemonSprites[1].Position)
		// スプライト無しは空配列(null にしない)
		require.NotNil(t, res.Data[1].PokemonSprites)
		require.Empty(t, res.Data[1].PokemonSprites)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestOpponentDeckController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		mockUsecase.EXPECT().FindByUserId(gomock.Any(), uid).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOpponentDeckController_Replace(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	path := MatchesPath + OpponentDecksPath

	put := func(c *OpponentDeck, secretKey string, body any) *httptest.ResponseRecorder {
		t.Helper()

		data, err := json.Marshal(body)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", path, bytes.NewBuffer(data))
		if secretKey != "" {
			setJWTAuthHeader(t, req, uid, secretKey)
		}
		c.router.ServeHTTP(w, req)

		return w
	}

	validBody := dto.OpponentDeckReplaceRequest{
		From: &dto.OpponentDeckSpecRequest{
			OpponentsDeckInfo: "ドラパ",
			PokemonSprites:    []*dto.PokemonSpriteRequest{},
		},
		To: &dto.OpponentDeckSpecRequest{
			OpponentsDeckInfo: "ドラパルトex",
			PokemonSprites: []*dto.PokemonSpriteRequest{
				{ID: "0887", Position: 1},
				{ID: "0006", Position: 2},
			},
		},
	}

	t.Run("正常系_置き換えた対戦の数を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		mockUsecase.EXPECT().Replace(gomock.Any(), uid, gomock.Any()).DoAndReturn(
			func(_ any, _ string, param *usecase.OpponentDeckReplaceParam) (int, error) {
				require.Equal(t, "ドラパ", param.From.OpponentsDeckInfo)
				require.Empty(t, param.From.PokemonSprites)
				require.Equal(t, "ドラパルトex", param.To.OpponentsDeckInfo)
				require.Len(t, param.To.PokemonSprites, 2)
				require.Equal(t, "0006", param.To.PokemonSprites[1].ID)
				require.Equal(t, uint(2), param.To.PokemonSprites[1].Position)
				return 3, nil
			})

		w := put(c, secretKey, validBody)

		var res dto.OpponentDeckReplaceResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 3, res.UpdatedCount)
	})

	t.Run("異常系_置き換え元が表記もスプライトも空なら400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckController(t)

		w := put(c, secretKey, dto.OpponentDeckReplaceRequest{
			From: &dto.OpponentDeckSpecRequest{OpponentsDeckInfo: ""},
			To:   validBody.To,
		})

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_置き換え先の表記が空白だけなら400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckController(t)

		w := put(c, secretKey, dto.OpponentDeckReplaceRequest{
			From: validBody.From,
			To:   &dto.OpponentDeckSpecRequest{OpponentsDeckInfo: "　 "},
		})

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_fromかtoが無ければ400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckController(t)

		w := put(c, secretKey, dto.OpponentDeckReplaceRequest{From: validBody.From})

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_表記が長すぎる・スプライトが3体以上なら400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckController(t)

		long := make([]rune, 64)
		for i := range long {
			long[i] = 'あ'
		}
		w := put(c, secretKey, dto.OpponentDeckReplaceRequest{
			From: validBody.From,
			To:   &dto.OpponentDeckSpecRequest{OpponentsDeckInfo: string(long)},
		})
		require.Equal(t, http.StatusBadRequest, w.Code)

		w = put(c, secretKey, dto.OpponentDeckReplaceRequest{
			From: validBody.From,
			To: &dto.OpponentDeckSpecRequest{
				OpponentsDeckInfo: "ドラパルトex",
				PokemonSprites: []*dto.PokemonSpriteRequest{
					{ID: "0887"}, {ID: "0006"}, {ID: "0025"},
				},
			},
		})
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_存在しないスプライトなら400を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		mockUsecase.EXPECT().Replace(gomock.Any(), uid, gomock.Any()).Return(0, apperror.ErrInvalidReference)

		w := put(c, secretKey, validBody)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		mockUsecase.EXPECT().Replace(gomock.Any(), uid, gomock.Any()).Return(0, errors.New(""))

		w := put(c, secretKey, validBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestOpponentDeckController(t)

		w := put(c, "", validBody)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
