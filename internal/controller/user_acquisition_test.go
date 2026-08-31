package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func setup4TestUserAcquisitionController(t *testing.T) (
	*UserAcquisition,
	*mock_usecase.MockUserAcquisitionInterface,
	string,
) {
	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockUserAcquisitionInterface(mockCtrl)

	r := gin.Default()
	c := NewUserAcquisition(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func newUserAcquisitionRequest(t *testing.T, body string, uid string, secretKey string) *http.Request {
	t.Helper()

	req, err := http.NewRequest("POST", UsersPath+UserAcquisitionPath, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if uid != "" {
		setJWTAuthHeader(t, req, uid, secretKey)
	}

	return req
}

func TestUserAcquisitionController(t *testing.T) {
	// レート制限はパッケージ変数として共有されるため、テストごとに異なるuidを使う
	t.Run("Record", func(t *testing.T) {
		t.Run("正常系_流入元をユースケースへ渡して204を返す", func(t *testing.T) {
			uid := "acquisition-ok-user"
			c, mockUsecase, secretKey := setup4TestUserAcquisitionController(t)

			mockUsecase.EXPECT().Record(gomock.Any(), uid, &usecase.UserAcquisitionRecordParam{
				Source:      "x",
				Medium:      "social",
				Campaign:    "howto_cta",
				Content:     "20260831a",
				Referrer:    "t.co",
				LandingPath: "/records/quick",
				LandingAt:   "2026-08-30T12:00:00.000Z",
			}).Return(nil)

			body := `{"source":"x","medium":"social","campaign":"howto_cta","content":"20260831a",` +
				`"referrer":"t.co","landing_path":"/records/quick","landing_at":"2026-08-30T12:00:00.000Z"}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionRequest(t, body, uid, secretKey))

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("正常系_判明しなかった項目がnullでも受け付ける", func(t *testing.T) {
			// Cookie を作る proxy 側は、取れなかった項目に null を入れる
			uid := "acquisition-null-user"
			c, mockUsecase, secretKey := setup4TestUserAcquisitionController(t)

			mockUsecase.EXPECT().Record(gomock.Any(), uid, &usecase.UserAcquisitionRecordParam{
				Source: "x",
			}).Return(nil)

			body := `{"source":"x","medium":null,"campaign":null,"content":null,"referrer":null,"landing_path":null,"landing_at":null}`
			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionRequest(t, body, uid, secretKey))

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _, _ := setup4TestUserAcquisitionController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionRequest(t, `{"source":"x"}`, "", ""))

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_JSONとして不正なボディなら400を返す", func(t *testing.T) {
			uid := "acquisition-badjson-user"
			c, _, secretKey := setup4TestUserAcquisitionController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionRequest(t, `not json`, uid, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			uid := "acquisition-error-user"
			c, mockUsecase, secretKey := setup4TestUserAcquisitionController(t)

			mockUsecase.EXPECT().Record(gomock.Any(), uid, gomock.Any()).Return(errors.New("failed to record"))

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionRequest(t, `{"source":"x"}`, uid, secretKey))

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})

		t.Run("異常系_同じユーザーの連投は429で止める", func(t *testing.T) {
			uid := "acquisition-ratelimit-user"
			c, mockUsecase, secretKey := setup4TestUserAcquisitionController(t)

			mockUsecase.EXPECT().Record(gomock.Any(), uid, gomock.Any()).Return(nil).AnyTimes()

			var lastCode int
			for i := 0; i < 20; i++ {
				w := httptest.NewRecorder()
				c.router.ServeHTTP(w, newUserAcquisitionRequest(t, `{"source":"x"}`, uid, secretKey))
				lastCode = w.Code
			}

			require.Equal(t, http.StatusTooManyRequests, lastCode)
		})
	})
}

func newUserAcquisitionSurveyRequest(t *testing.T, body string, uid string, secretKey string) *http.Request {
	t.Helper()

	req, err := http.NewRequest("POST", UsersPath+UserAcquisitionSurveyPath, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if uid != "" {
		setJWTAuthHeader(t, req, uid, secretKey)
	}

	return req
}

func TestUserAcquisitionSurveyController(t *testing.T) {
	// レート制限はパッケージ変数として共有されるため、テストごとに異なるuidを使う
	t.Run("AnswerSurvey", func(t *testing.T) {
		t.Run("正常系_回答をユースケースへ渡して204を返す", func(t *testing.T) {
			uid := "survey-ok-user"
			c, mockUsecase, secretKey := setup4TestUserAcquisitionController(t)

			mockUsecase.EXPECT().AnswerSurvey(gomock.Any(), uid, "friend").Return(nil)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionSurveyRequest(t, `{"answer":"friend"}`, uid, secretKey))

			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("異常系_allowlist外の回答は400を返す", func(t *testing.T) {
			// UIの4択以外が届くのはUI外からのリクエスト。黙って捨てると
			// UI側の実装ミスにも気づけないため 400 で返す
			uid := "survey-badanswer-user"
			c, _, secretKey := setup4TestUserAcquisitionController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionSurveyRequest(t, `{"answer":"instagram"}`, uid, secretKey))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
			c, _, _ := setup4TestUserAcquisitionController(t)

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionSurveyRequest(t, `{"answer":"x"}`, "", ""))

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
			uid := "survey-error-user"
			c, mockUsecase, secretKey := setup4TestUserAcquisitionController(t)

			mockUsecase.EXPECT().AnswerSurvey(gomock.Any(), uid, "x").Return(errors.New("failed to save"))

			w := httptest.NewRecorder()
			c.router.ServeHTTP(w, newUserAcquisitionSurveyRequest(t, `{"answer":"x"}`, uid, secretKey))

			require.Equal(t, http.StatusInternalServerError, w.Code)
		})

		t.Run("異常系_同じユーザーの連投は429で止める", func(t *testing.T) {
			uid := "survey-ratelimit-user"
			c, mockUsecase, secretKey := setup4TestUserAcquisitionController(t)

			mockUsecase.EXPECT().AnswerSurvey(gomock.Any(), uid, "x").Return(nil).AnyTimes()

			var lastCode int
			for i := 0; i < 20; i++ {
				w := httptest.NewRecorder()
				c.router.ServeHTTP(w, newUserAcquisitionSurveyRequest(t, `{"answer":"x"}`, uid, secretKey))
				lastCode = w.Code
			}

			require.Equal(t, http.StatusTooManyRequests, lastCode)
		})
	})
}
