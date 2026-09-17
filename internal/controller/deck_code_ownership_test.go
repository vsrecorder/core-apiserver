package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

// デッキコードの deck_id は認可ミドルウェアを通らないため、usecase が所有者を検証し
// 他人・存在しないデッキは ErrRecordNotFound を返す。コントローラはそれを 404 にする。
func TestDeckCodeController_Create_ReferenceOwnership(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("異常系_参照先のデッキが他人または存在しなければ404を返す", func(t *testing.T) {
		c, _, _, secretKey, _ := setup4TestDeckCodeController(t, stubDeckCodeUsecase{err: apperror.ErrRecordNotFound})

		b, err := json.Marshal(dto.DeckCodeCreateRequest{DeckId: "01HD7Y3K8D6FDHMHTZ2GT41TD1", Code: "5dbFbk-uBwjqP-VVk5Vv"})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", DeckCodesPath, strings.NewReader(string(b)))
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})
}
