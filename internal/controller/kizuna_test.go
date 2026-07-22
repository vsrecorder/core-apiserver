package controller

import (
	"context"
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
)

func setup4TestKizunaController(t *testing.T) (*Kizuna, *mock_usecase.MockKizunaInterface, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockKizunaInterface(mockCtrl)

	r := gin.Default()
	c := NewKizuna(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase, secretKey
}

func TestKizunaController_GetByUserId(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_本人なら全デッキのきずなLv.を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestKizunaController(t)

		mockUsecase.EXPECT().GetKizuna(context.Background(), uid).
			Return(entity.NewKizuna(uid, []*entity.KizunaDeck{
				{
					DeckId: "deck-01",
					Level:  178,
					Metrics: []*entity.KizunaMetric{
						{Key: entity.KizunaMetricLoyalty, Weight: 20, Value: 0.79, Points: 49, MaxPoints: 62},
					},
				},
			}), nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+KizunaPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.KizunaResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		require.Equal(t, uid, res.UserId)
		// 上限をクライアントに直書きさせないため、レスポンスに含めている
		require.Equal(t, entity.KizunaMaxLevel, res.MaxLevel)
		require.Len(t, res.Decks, 1)
		require.Equal(t, "deck-01", res.Decks[0].DeckId)
		require.Equal(t, 178, res.Decks[0].Level)
		require.Equal(t, "loyalty", res.Decks[0].Metrics[0].Key)
	})

	t.Run("正常系_デッキが1つも無くてもdecksはnullではなく空配列を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestKizunaController(t)

		mockUsecase.EXPECT().GetKizuna(context.Background(), uid).
			Return(entity.NewKizuna(uid, []*entity.KizunaDeck{}), nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+KizunaPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"decks":[]`)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestKizunaController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+KizunaPath, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_他人のきずなLv.は403で見せない", func(t *testing.T) {
		c, _, secretKey := setup4TestKizunaController(t)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+KizunaPath, nil)
		setJWTAuthHeader(t, req, "other-user", secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_レコードが見つからなければ404を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestKizunaController(t)

		mockUsecase.EXPECT().GetKizuna(context.Background(), uid).
			Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+KizunaPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_想定外のエラーなら500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestKizunaController(t)

		mockUsecase.EXPECT().GetKizuna(context.Background(), uid).
			Return(nil, errors.New("unexpected"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", UsersPath+"/"+uid+KizunaPath, nil)
		setJWTAuthHeader(t, req, uid, secretKey)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
