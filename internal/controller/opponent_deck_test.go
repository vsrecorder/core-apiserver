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

func TestOpponentDeckController_GetMatches(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	path := MatchesPath + OpponentDeckMatchesPath

	get := func(c *OpponentDeck, secretKey string, query string) *httptest.ResponseRecorder {
		t.Helper()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+query, nil)
		if secretKey != "" {
			setJWTAuthHeader(t, req, uid, secretKey)
		}
		c.router.ServeHTTP(w, req)

		return w
	}

	t.Run("正常系_組み合わせの対戦を記録の見出しと対局付きで返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		matches := []*entity.OpponentDeckMatch{
			{
				MatchId:    "01K6MATCH1",
				RecordId:   "01K6RECORD1",
				EventDate:  "2026-09-28",
				EventType:  "official",
				EventTitle: "シティリーグ 東京",
				DeckName:   "サーナイト",
				BO3Flg:     true,
				VictoryFlg: true,
				Games: []*entity.OpponentDeckMatchGame{
					{GoFirst: true, WinningFlg: true},
					{GoFirst: false, WinningFlg: false},
					{GoFirst: true, WinningFlg: true},
				},
			},
			{
				MatchId:           "01K6MATCH2",
				RecordId:          "01K6RECORD2",
				DefaultVictoryFlg: true,
				Games:             []*entity.OpponentDeckMatchGame{},
			},
		}
		mockUsecase.EXPECT().FindMatches(gomock.Any(), uid, gomock.Any()).DoAndReturn(
			func(_ any, _ string, param *usecase.OpponentDeckSpecParam) ([]*entity.OpponentDeckMatch, error) {
				require.Equal(t, "ドラパルト ex", param.OpponentsDeckInfo)
				// 2体目だけの指定は position 2 のまま渡す
				require.Len(t, param.PokemonSprites, 1)
				require.Equal(t, "0887", param.PokemonSprites[0].ID)
				require.Equal(t, uint(2), param.PokemonSprites[0].Position)
				return matches, nil
			})

		w := get(c, secretKey, "?opponents_deck_info=%E3%83%89%E3%83%A9%E3%83%91%E3%83%AB%E3%83%88+ex&pokemon_sprite_id_2=0887")

		var res dto.OpponentDeckMatchesGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Len(t, res.Data, 2)
		require.Equal(t, "01K6MATCH1", res.Data[0].ID)
		require.Equal(t, "01K6RECORD1", res.Data[0].RecordId)
		require.Equal(t, "2026-09-28", res.Data[0].EventDate)
		require.Equal(t, "official", res.Data[0].EventType)
		require.Equal(t, "シティリーグ 東京", res.Data[0].EventTitle)
		require.Equal(t, "サーナイト", res.Data[0].DeckName)
		require.True(t, res.Data[0].BO3Flg)
		require.True(t, res.Data[0].VictoryFlg)
		require.Len(t, res.Data[0].Games, 3)
		require.False(t, res.Data[0].Games[1].WinningFlg)
		require.True(t, res.Data[1].DefaultVictoryFlg)
		// 対局の無い対戦は空配列(null にしない)
		require.NotNil(t, res.Data[1].Games)
		require.Empty(t, res.Data[1].Games)
	})

	t.Run("正常系_スプライトだけの組み合わせも引ける", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		mockUsecase.EXPECT().FindMatches(gomock.Any(), uid, gomock.Any()).DoAndReturn(
			func(_ any, _ string, param *usecase.OpponentDeckSpecParam) ([]*entity.OpponentDeckMatch, error) {
				require.Equal(t, "", param.OpponentsDeckInfo)
				require.Len(t, param.PokemonSprites, 2)
				require.Equal(t, uint(1), param.PokemonSprites[0].Position)
				require.Equal(t, uint(2), param.PokemonSprites[1].Position)
				return []*entity.OpponentDeckMatch{}, nil
			})

		w := get(c, secretKey, "?pokemon_sprite_id_1=0887&pokemon_sprite_id_2=0006")

		var res dto.OpponentDeckMatchesGetResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, res.Data)
		require.Empty(t, res.Data)
	})

	t.Run("異常系_表記もスプライトも無ければ400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckController(t)

		w := get(c, secretKey, "?opponents_deck_info=")

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_表記が長すぎる・スプライトIDの形式が違えば400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestOpponentDeckController(t)

		long := make([]rune, 64)
		for i := range long {
			long[i] = 'a'
		}
		w := get(c, secretKey, "?opponents_deck_info="+string(long))
		require.Equal(t, http.StatusBadRequest, w.Code)

		w = get(c, secretKey, "?opponents_deck_info=x&pokemon_sprite_id_1=%27%3B")
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestOpponentDeckController(t)

		mockUsecase.EXPECT().FindMatches(gomock.Any(), uid, gomock.Any()).Return(nil, errors.New(""))

		w := get(c, secretKey, "?opponents_deck_info=x")

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestOpponentDeckController(t)

		w := get(c, "", "?opponents_deck_info=x")

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
