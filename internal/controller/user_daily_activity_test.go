package controller

import (
	"encoding/json"
	"errors"
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
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

func setup4TestUserDailyActivityController(t *testing.T) (
	*UserDailyActivity,
	*mock_usecase.MockUserDailyActivityInterface,
	string,
) {
	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockUserDailyActivityInterface(mockCtrl)

	r := gin.Default()
	c := NewUserDailyActivity(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func newActivityRequest(t *testing.T, body string, uid string, secretKey string) *http.Request {
	t.Helper()

	req, err := http.NewRequest("POST", UsersPath+UserDailyActivityPath, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	setJWTAuthHeader(t, req, uid, secretKey)

	return req
}

func TestUserDailyActivityController_Record(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_受け取ったカテゴリをそのままユースケースへ渡し204を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestUserDailyActivityController(t)

		categories := []string{
			entity.UserDailyActivityCategoryVisit,
			entity.UserDailyActivityCategoryReview,
		}

		mockUsecase.EXPECT().Record(gomock.Any(), uid, categories).Return(nil)

		body, err := json.Marshal(dto.UserDailyActivityRequest{Categories: categories})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c.router.ServeHTTP(w, newActivityRequest(t, string(body), uid, secretKey))

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("正常系_bodyが空ならvisitとして扱う", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestUserDailyActivityController(t)

		mockUsecase.EXPECT().Record(
			gomock.Any(), uid, []string{entity.UserDailyActivityCategoryVisit},
		).Return(nil)

		w := httptest.NewRecorder()
		c.router.ServeHTTP(w, newActivityRequest(t, "", uid, secretKey))

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("正常系_bodyが壊れていてもvisitとして扱う", func(t *testing.T) {
		// 計測は非クリティカル処理のため、パースできないbodyでも既定値へ丸めて受け付ける
		c, mockUsecase, secretKey := setup4TestUserDailyActivityController(t)

		mockUsecase.EXPECT().Record(
			gomock.Any(), uid, []string{entity.UserDailyActivityCategoryVisit},
		).Return(nil)

		w := httptest.NewRecorder()
		c.router.ServeHTTP(w, newActivityRequest(t, "{not-json", uid, secretKey))

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_既知のカテゴリが1つも無ければ400を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestUserDailyActivityController(t)

		mockUsecase.EXPECT().Record(gomock.Any(), uid, []string{"unknown"}).
			Return(apperror.ErrNoKnownActivityCategory)

		w := httptest.NewRecorder()
		c.router.ServeHTTP(w, newActivityRequest(t, `{"categories":["unknown"]}`, uid, secretKey))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestUserDailyActivityController(t)

		mockUsecase.EXPECT().Record(gomock.Any(), uid, gomock.Any()).Return(errors.New(""))

		w := httptest.NewRecorder()
		c.router.ServeHTTP(w, newActivityRequest(t, `{"categories":["visit"]}`, uid, secretKey))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("正常系_users配下の既存ルートと同じルータに同居できる", func(t *testing.T) {
		// /users/activity は静的セグメントのため、/users/:id 系と同じルータへ登録すると
		// 登録の順序や組み合わせによってはginが起動時にpanicする。
		// main.go と同じ並びで登録しても事故らないことをここで担保する。
		gin.SetMode(gin.TestMode)

		noop := func(*gin.Context) {}

		require.NotPanics(t, func() {
			r := gin.New()
			r.POST(UsersPath, noop)                       // user.go: ユーザ作成
			r.PUT(UsersPath+"/:id", noop)                 // user.go: ユーザ更新
			r.DELETE(UsersPath+"/:id", noop)              // user.go: 退会
			r.GET(UsersPath+"/:id"+StreakPath, noop)      // streak.go: ストリーク取得
			r.POST(UsersPath+UserDailyActivityPath, noop) // 本コントローラ
		})
	})

	t.Run("異常系_未認証は401を返す", func(t *testing.T) {
		// uidをトークンから取る以上、認証を通していない書き込みは受け付けない
		c, mockUsecase, _ := setup4TestUserDailyActivityController(t)

		mockUsecase.EXPECT().Record(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		req, err := http.NewRequest("POST", UsersPath+UserDailyActivityPath, strings.NewReader(`{"categories":["visit"]}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
