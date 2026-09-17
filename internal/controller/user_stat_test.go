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

func setup4TestUserStatController(t *testing.T) (
	*UserStat,
	*mock_usecase.MockUserStatInterface,
	*mock_usecase.MockUserStatHistoryInterface,
	string,
) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockUserStatInterface(mockCtrl)
	mockHistoryUsecase := mock_usecase.NewMockUserStatHistoryInterface(mockCtrl)

	r := gin.Default()
	c := NewUserStat(r, mockUsecase, mockHistoryUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, mockHistoryUsecase, secretKey
}

func TestUserStatController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("GetByUserId", func(t *testing.T) {
		t.Run("正常系_集計条件をユースケースへ渡して統計を返す", func(t *testing.T) {
			c, mockUsecase, _, secretKey := setup4TestUserStatController(t)

			stat := entity.NewUserStat(uid, 5, 2, 1, 1, 10, 6, 4, 0.6)

			mockUsecase.EXPECT().GetUserStat(gomock.Any(), uid, "", "2026-07", "sv11", "", "", uint(0), false).Return(stat, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"?year_month=2026-07&environment_id=sv11", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("正常系_exclude_default_matchesをユースケースへ渡し応答にも載せる", func(t *testing.T) {
			c, mockUsecase, _, secretKey := setup4TestUserStatController(t)

			stat := entity.NewUserStat(uid, 5, 2, 1, 1, 8, 5, 3, 0.625)

			mockUsecase.EXPECT().GetUserStat(gomock.Any(), uid, "", "2026-07", "", "", "", uint(0), true).Return(stat, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"?year_month=2026-07&exclude_default_matches=true", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"exclude_default_matches":true`)
		})

		// false は omitempty で落とさない。「不戦も含めて数えた」という結果そのものなので、
		// 落とすと受け取り側が「指定しなかった」と区別できない
		t.Run("正常系_不戦を含めたときも応答にfalseを載せる", func(t *testing.T) {
			c, mockUsecase, _, secretKey := setup4TestUserStatController(t)

			stat := entity.NewUserStat(uid, 5, 2, 1, 1, 10, 6, 4, 0.6)

			mockUsecase.EXPECT().GetUserStat(gomock.Any(), uid, "", "2026-07", "", "", "", uint(0), false).Return(stat, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"?year_month=2026-07&exclude_default_matches=false", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"exclude_default_matches":false`)
		})

		t.Run("異常系_exclude_default_matchesが真偽値でなければ400を返す", func(t *testing.T) {
			c, _, _, secretKey := setup4TestUserStatController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"?exclude_default_matches=abc", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_year_monthの形式が不正なら400を返す", func(t *testing.T) {
			c, _, _, secretKey := setup4TestUserStatController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"?year_month=abc", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_該当なしはErrRecordNotFoundから404を返す", func(t *testing.T) {
			c, mockUsecase, _, secretKey := setup4TestUserStatController(t)

			mockUsecase.EXPECT().GetUserStat(gomock.Any(), uid, "", "", "", "", "", uint(0), false).Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, mockUsecase, _, secretKey := setup4TestUserStatController(t)

			mockUsecase.EXPECT().GetUserStat(gomock.Any(), uid, "", "", "", "", "", uint(0), false).Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath, nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

	t.Run("GetHistoryByUserId", func(t *testing.T) {
		t.Run("正常系_期間指定で月次の履歴を返す", func(t *testing.T) {
			c, _, mockHistoryUsecase, secretKey := setup4TestUserStatController(t)

			history := []*entity.UserStatMonthly{entity.NewUserStatMonthly("2026-06", 4, 3, 1, 0.75)}

			mockHistoryUsecase.EXPECT().GetUserStatHistory(gomock.Any(), uid, "6months", "", "", uint(0), false).Return(history, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history?period=6months", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("正常系_exclude_default_matchesをユースケースへ渡す", func(t *testing.T) {
			c, _, mockHistoryUsecase, secretKey := setup4TestUserStatController(t)

			history := []*entity.UserStatMonthly{entity.NewUserStatMonthly("2026-06", 3, 2, 1, 0.6666666666666666)}

			mockHistoryUsecase.EXPECT().GetUserStatHistory(gomock.Any(), uid, "6months", "", "", uint(0), true).Return(history, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history?period=6months&exclude_default_matches=true", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"exclude_default_matches":true`)
		})

		t.Run("異常系_exclude_default_matchesが真偽値でなければ400を返す", func(t *testing.T) {
			c, _, _, secretKey := setup4TestUserStatController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history?exclude_default_matches=abc", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_未定義のperiodなら400を返す", func(t *testing.T) {
			c, _, _, secretKey := setup4TestUserStatController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history?period=1year", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			c, _, mockHistoryUsecase, secretKey := setup4TestUserStatController(t)

			mockHistoryUsecase.EXPECT().GetUserStatHistory(gomock.Any(), uid, "3months", "", "", uint(0), false).Return(nil, errors.New(""))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history", nil)
			setJWTAuthHeader(t, req, uid, secretKey)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})
	})

}

func TestUserStatController_GetByUserId_Authorization(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _, _ := setup4TestUserStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人の戦績は403で見せない", func(t *testing.T) {
		c, _, _, secretKey := setup4TestUserStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath, nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestUserStatController_GetHistoryByUserId_Authorization(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _, _ := setup4TestUserStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history", nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人の戦績の推移は403で見せない", func(t *testing.T) {
		c, _, _, secretKey := setup4TestUserStatController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+UserStatsPath+"/history", nil)
		setJWTAuthHeader(t, req, "KBp7roRDZobZg1t0OPzFR1kvLeO2", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}
