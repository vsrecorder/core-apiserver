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

// 対戦結果の record_id / deck_id / deck_code_id は認可ミドルウェアを通らないため、
// usecase が所有者を検証し他人・存在しないものは ErrRecordNotFound を返す。コントローラはそれを 404 にする。
func TestMatchController_ReferenceOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	newRequestBody := func(t *testing.T, recordId string) string {
		t.Helper()

		b, err := json.Marshal(dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId: recordId,
				Games:    []*dto.GameRequest{{GoFirst: false, WinningFlg: false}},
			},
		})
		require.NoError(t, err)

		return string(b)
	}

	t.Run("Create_異常系_参照先の記録が他人または存在しなければ404を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, _, mockUsecase := setup4TestMatchController(t, r)

		recordId, _ := generateId()
		mockUsecase.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/matches", strings.NewReader(newRequestBody(t, recordId)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Update_異常系_参照先の記録が他人または存在しなければ404を返す", func(t *testing.T) {
		r := gin.Default()
		c, mockMatchRepository, _, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()
		recordId, _ := generateId()

		// MatchUpdateAuthorizationMiddleware が本人確認のために参照する
		mockMatchRepository.EXPECT().FindById(gomock.Any(), id).Return(&entity.Match{ID: id, UserId: uid}, nil)
		mockUsecase.EXPECT().Update(gomock.Any(), id, gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("PUT", "/matches/"+id, strings.NewReader(newRequestBody(t, recordId)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})
}
