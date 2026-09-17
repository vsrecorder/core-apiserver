package controller

import (
	"encoding/json"
	"fmt"
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

// 記録の deck_id / deck_code_id は認可ミドルウェアを通らないため、usecase が所有者を検証し
// 他人・存在しないものは ErrRecordNotFound を返す。コントローラはそれを 404 にする。
func TestRecordController_ReferenceOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	newRequestBody := func(t *testing.T) string {
		t.Helper()

		b, err := json.Marshal(dto.RecordCreateRequest{
			RecordRequest: dto.RecordRequest{
				EventDate:       testControllerEventDate,
				OfficialEventId: 10000,
				DeckId:          deckId,
			},
		})
		require.NoError(t, err)

		return string(b)
	}

	t.Run("Create_異常系_参照先のデッキが他人または存在しなければ404を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, mockUsecase := setup4TestRecordController(t, r)

		mockUsecase.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", RecordsPath, strings.NewReader(newRequestBody(t)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Update_異常系_参照先のデッキが他人または存在しなければ404を返す", func(t *testing.T) {
		r := gin.Default()
		c, mockRepository, mockUsecase := setup4TestRecordController(t, r)

		id, err := generateId()
		require.NoError(t, err)

		// RecordUpdateAuthorizationMiddleware が本人確認のために参照する
		mockRepository.EXPECT().FindById(gomock.Any(), id).Return(&entity.Record{ID: id, UserId: uid}, nil)
		mockUsecase.EXPECT().Update(gomock.Any(), id, gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("PUT", RecordsPath+"/"+id, strings.NewReader(newRequestBody(t)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})
}

// deck_id で絞った記録一覧は、認証済みユーザー自身の記録に限る。
// デッキIDは公開情報から誰でも知り得るため、uid を渡さないと他人の非公開記録まで返ってしまう。
func TestRecordController_GetByUserId_DeckFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	t.Run("正常系_deck_id指定時は本人の記録に限って取得する", func(t *testing.T) {
		r := gin.Default()
		c, _, mockUsecase := setup4TestRecordController(t, r)

		limit := 10
		offset := 0
		records := []*entity.Record{{ID: "01HD7Y3K8D6FDHMHTZ2GT41TR1", UserId: uid, DeckId: deckId}}

		// usecase には認証済みの uid が渡ること(deck_id だけで引かない)
		mockUsecase.EXPECT().FindByDeckId(gomock.Any(), uid, deckId, limit, offset, "").Return(records, nil)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?deck_id=%s&limit=%d&offset=%d", deckId, limit, offset), nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("正常系_カーソル指定でもdeck_id指定時は本人の記録に限って取得する", func(t *testing.T) {
		r := gin.Default()
		c, _, mockUsecase := setup4TestRecordController(t, r)

		cursorEventDate := testControllerEventDate
		cursorCreatedAt := testControllerEventDate

		mockUsecase.EXPECT().FindByDeckIdOnCursor(gomock.Any(), uid, deckId, 10, cursorEventDate, cursorCreatedAt, "").Return([]*entity.Record{}, nil)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", fmt.Sprintf(RecordsPath+"?deck_id=%s&cursor=%s", deckId, encodeTestCursor(cursorEventDate, cursorCreatedAt)), nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}
