package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

// 形式は正しいが存在しないスプライトIDは、保存時の外部キー違反が
// apperror.ErrInvalidReference として上がってくる。DB エラーの 500 ではなく 400 で返す。
func TestInvalidReference_ReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	t.Run("Deck_Create_異常系_参照先の無いスプライトは400を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, _, mockUsecase := setup4TestDeckController(t, r)

		mockUsecase.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInvalidReference)

		b, err := json.Marshal(dto.DeckCreateRequest{Name: "test", PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "9999", Position: 1}}})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", DecksPath, strings.NewReader(string(b)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Deck_Update_異常系_参照先の無いスプライトは400を返す", func(t *testing.T) {
		r := gin.Default()
		c, mockDeckRepository, _, mockUsecase := setup4TestDeckController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		// DeckUpdateAuthorizationMiddleware が本人確認のために参照する
		mockDeckRepository.EXPECT().FindById(gomock.Any(), id).Return(&entity.Deck{ID: id, UserId: uid}, nil)
		mockUsecase.EXPECT().Update(gomock.Any(), id, gomock.Any()).Return(nil, apperror.ErrInvalidReference)

		b, err := json.Marshal(dto.DeckUpdateRequest{Name: "test", PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "9999", Position: 1}}})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("PUT", DecksPath+"/"+id, strings.NewReader(string(b)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Match_Create_異常系_参照先の無いスプライトは400を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, _, mockUsecase := setup4TestMatchController(t, r)

		recordId, _ := generateId()
		mockUsecase.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInvalidReference)

		b, err := json.Marshal(dto.MatchCreateRequest{MatchRequest: dto.MatchRequest{
			RecordId:       recordId,
			Games:          []*dto.GameRequest{{GoFirst: false, WinningFlg: false}},
			PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "9999", Position: 1}},
		}})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/matches", strings.NewReader(string(b)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
