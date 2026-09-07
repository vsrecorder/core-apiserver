package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
)

// setup4TestMatchSummaryController は認証が必須の GET /matches/summary 用に、
// JWTの署名鍵まで用意したコントローラを組み立てる。
func setup4TestMatchSummaryController(t *testing.T) (*Match, *mock_usecase.MockMatchInterface, string) {
	t.Helper()

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	r := gin.Default()
	c, _, _, mockUsecase := setup4TestMatchController(t, r)

	return c, mockUsecase, secretKey
}

func test_MatchController_GetSummaries(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	recordId1 := "01HD7Y3K8D6FDHMHTZ2GT41TR1"
	recordId2 := "01HD7Y3K8D6FDHMHTZ2GT41TR2"

	t.Run("正常系_指定した記録の集計をまとめて返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestMatchSummaryController(t)

		// 認可はusecase(のリポジトリ)がuidで絞るため、トークンのuidが渡ること
		mockUsecase.EXPECT().FindSummariesByRecordIds(
			gomock.Any(), uid, []string{recordId1, recordId2},
		).Return([]*entity.MatchSummary{
			entity.NewMatchSummary(recordId1, 5, 3, 1, 1, false, true),
			entity.NewMatchSummary(recordId2, 0, 0, 0, 0, false, false),
		}, nil)

		w := httptest.NewRecorder()
		req, err := http.NewRequest(
			"GET", MatchesPath+MatchesSummaryPath+"?record_ids="+recordId1+","+recordId2, nil,
		)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.MatchGetSummariesResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		require.Len(t, res.Summaries, 2)

		require.Equal(t, recordId1, res.Summaries[0].RecordId)
		require.Equal(t, 5, res.Summaries[0].Total)
		require.Equal(t, 3, res.Summaries[0].Wins)
		require.Equal(t, 1, res.Summaries[0].Losses)
		require.Equal(t, 1, res.Summaries[0].Draws)
		require.False(t, res.Summaries[0].HasGroupMatch)
		require.True(t, res.Summaries[0].HasBo3)

		require.Equal(t, recordId2, res.Summaries[1].RecordId)
		require.Equal(t, 0, res.Summaries[1].Total)
	})

	// 他人の記録が混ざっても403/404にはせず、返せるものだけを返す。
	// 1件でも他人のIDが混ざるとページ全体が表示できなくなるのを避けるため。
	t.Run("正常系_他人の記録は結果から除外され残りが返る", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestMatchSummaryController(t)

		othersRecordId := "01HD7Y3K8D6FDHMHTZ2GT41TR9"

		mockUsecase.EXPECT().FindSummariesByRecordIds(
			gomock.Any(), uid, []string{recordId1, othersRecordId},
		).Return([]*entity.MatchSummary{
			entity.NewMatchSummary(recordId1, 1, 1, 0, 0, false, false),
		}, nil)

		w := httptest.NewRecorder()
		req, err := http.NewRequest(
			"GET", MatchesPath+MatchesSummaryPath+"?record_ids="+recordId1+","+othersRecordId, nil,
		)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.MatchGetSummariesResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		require.Len(t, res.Summaries, 1)
		require.Equal(t, recordId1, res.Summaries[0].RecordId)
	})

	t.Run("正常系_record_ids未指定なら空配列を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestMatchSummaryController(t)

		mockUsecase.EXPECT().FindSummariesByRecordIds(gomock.Any(), uid, []string{}).
			Return([]*entity.MatchSummary{}, nil)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", MatchesPath+MatchesSummaryPath, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var res dto.MatchGetSummariesResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		require.Empty(t, res.Summaries)
		// nullではなく[]で返ること(webapp側で分岐させないため)
		require.Contains(t, w.Body.String(), `"summaries":[]`)
	})

	t.Run("正常系_重複した記録IDは1件にまとめて渡される", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestMatchSummaryController(t)

		mockUsecase.EXPECT().FindSummariesByRecordIds(gomock.Any(), uid, []string{recordId1}).
			Return([]*entity.MatchSummary{
				entity.NewMatchSummary(recordId1, 1, 0, 1, 0, true, false),
			}, nil)

		w := httptest.NewRecorder()
		req, err := http.NewRequest(
			"GET", MatchesPath+MatchesSummaryPath+"?record_ids="+recordId1+","+recordId1, nil,
		)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_上限を超える件数を指定したら400を返す", func(t *testing.T) {
		c, _, secretKey := setup4TestMatchSummaryController(t)

		// usecaseは呼ばれない(EXPECTを設定していないため、呼ばれたらgomockが落とす)
		recordIds := make([]string, 0, helper.MaxRecordIds+1)
		for i := 0; i <= helper.MaxRecordIds; i++ {
			recordIds = append(recordIds, "01HD7Y3K8D6FDHMHTZ2GT41T"+strconv.Itoa(i))
		}

		w := httptest.NewRecorder()
		req, err := http.NewRequest(
			"GET",
			MatchesPath+MatchesSummaryPath+"?record_ids="+strings.Join(recordIds, ","),
			nil,
		)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_未認証なら401を返す", func(t *testing.T) {
		c, _, _ := setup4TestMatchSummaryController(t)

		w := httptest.NewRecorder()
		req, err := http.NewRequest(
			"GET", MatchesPath+MatchesSummaryPath+"?record_ids="+recordId1, nil,
		)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("異常系_usecaseが失敗したら500を返す", func(t *testing.T) {
		c, mockUsecase, secretKey := setup4TestMatchSummaryController(t)

		mockUsecase.EXPECT().FindSummariesByRecordIds(gomock.Any(), uid, []string{recordId1}).
			Return(nil, errors.New("unexpected"))

		w := httptest.NewRecorder()
		req, err := http.NewRequest(
			"GET", MatchesPath+MatchesSummaryPath+"?record_ids="+recordId1, nil,
		)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
