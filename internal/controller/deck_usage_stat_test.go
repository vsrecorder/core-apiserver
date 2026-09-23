package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestDeckUsageStatController(t *testing.T) (*DeckUsageStat, *mock_usecase.MockDeckUsageStatInterface, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockDeckUsageStatInterface(mockCtrl)

	r := gin.Default()
	c := NewDeckUsageStat(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestDeckUsageStatController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_本人なら集計条件を渡してデッキ使用統計を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckUsageStat(gomock.Any(), uid, "", "2026-07", "", "", "", uint(0), true, false).
			Return(&entity.DeckUsageStat{}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath+"?year_month=2026-07&all_time=true", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	// 不戦勝/不戦敗の除外はクエリで受け取り、そのまま usecase に渡す(既定は含める)
	t.Run("正常系_exclude_default_matchesを渡すと除外指定でユースケースを呼ぶ", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckUsageStat(gomock.Any(), uid, "", "", "", "", "", uint(0), true, true).
			Return(&entity.DeckUsageStat{}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath+"?all_time=true&exclude_default_matches=true", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_exclude_default_matchesが真偽値でなければ400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath+"?exclude_default_matches=yes", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人の統計は403を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_該当なしはErrRecordNotFoundから404を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckUsageStat(gomock.Any(), uid, "", "", "", "", "", uint(0), false, false).
			Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckUsageStat(gomock.Any(), uid, "", "", "", "", "", uint(0), false, false).
			Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+DeckUsageStatsPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestDeckUsageStatController_GetDeckCodeUsageByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	path := UsersPath + "/" + uid + DeckCodeUsageStatsPath

	t.Run("正常系_本人ならデッキIDと除外指定を渡してバージョン別の成績を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckCodeUsageStat(gomock.Any(), uid, deckId, true).
			Return(&entity.DeckCodeUsageStat{
				UserId: uid,
				DeckId: deckId,
				DeckCodes: []*entity.DeckCodeUsage{
					entity.NewDeckCodeUsage("01HD7Y3K8D6FDHMHTZ2GT41TC1", 5, 3, 1),
				},
				UnassignedCount: 2,
			}, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+"?deck_id="+deckId+"&exclude_default_matches=true", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{
			"user_id": "`+uid+`",
			"deck_id": "`+deckId+`",
			"deck_codes": [
				{"deck_code_id": "01HD7Y3K8D6FDHMHTZ2GT41TC1", "count": 5, "wins": 3, "losses": 1, "draws": 1, "win_rate": 0.75}
			],
			"unassigned_count": 2
		}`, w.Body.String())
	})

	t.Run("異常系_deck_idが無ければ400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_deck_idが列幅を超えれば400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+"?deck_id="+deckId+"X", nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+"?deck_id="+deckId, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人の成績は403を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestDeckUsageStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+"?deck_id="+deckId, nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestDeckUsageStatController(t)

		mockUsecase.EXPECT().GetDeckCodeUsageStat(gomock.Any(), uid, deckId, false).
			Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path+"?deck_id="+deckId, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
