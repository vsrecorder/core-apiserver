package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func setupMock4TestMatchController(t *testing.T) (*mock_repository.MockMatchInterface, *mock_usecase.MockMatchInterface) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockMatchInterface(mockCtrl)
	mockUsecase := mock_usecase.NewMockMatchInterface(mockCtrl)

	return mockRepository, mockUsecase
}

func setup4TestMatchController(t *testing.T, r *gin.Engine) (
	*Match,
	*mock_usecase.MockMatchInterface,
) {
	authDisable := true
	mockRepository, mockUsecase := setupMock4TestMatchController(t)

	c := NewMatch(r, mockRepository, mockUsecase)
	c.RegisterRoute("", authDisable)

	return c, mockUsecase
}

func TestMatchController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetById":       test_MatchController_GetById,
		"GetByRecordId": test_MatchController_GetByRecordId,
		"Create":        test_MatchController_Create,
		"Update":        test_MatchController_Update,
		"Delete":        test_MatchController_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_MatchController_GetById(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()
		createdAt := time.Now().UTC().Truncate(0)
		recordId, _ := generateId()
		deckId := ""
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		match := &entity.Match{
			ID:              id,
			CreatedAt:       createdAt,
			RecordId:        recordId,
			DeckId:          deckId,
			UserId:          uid,
			OpponentsUserId: "",
		}

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(match, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_MatchController_GetByRecordId(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()
		createdAt := time.Now().UTC().Truncate(0)
		recordId, _ := generateId()
		deckId := ""
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		matches := []*entity.Match{
			{
				ID:              id,
				CreatedAt:       createdAt,
				RecordId:        recordId,
				DeckId:          deckId,
				UserId:          uid,
				OpponentsUserId: "",
			},
		}

		mockUsecase.EXPECT().FindByRecordId(context.Background(), recordId).Return(matches, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records/"+recordId+"/matches", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByRecordIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, recordId, res.RecordId)
		require.Equal(t, uid, res.UserId)
		require.Equal(t, len(matches), len(res.Matches))
		require.Equal(t, id, res.Matches[0].ID)
		require.Equal(t, createdAt, res.Matches[0].CreatedAt)
		require.Equal(t, recordId, res.Matches[0].RecordId)
		require.Equal(t, uid, res.Matches[0].UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		recordId, _ := generateId()

		mockUsecase.EXPECT().FindByRecordId(context.Background(), recordId).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records/"+recordId+"/matches", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		recordId, _ := generateId()

		mockUsecase.EXPECT().FindByRecordId(context.Background(), recordId).Return(nil, errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records/"+recordId+"/matches", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_MatchController_Create(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		createdAt := time.Now().UTC().Truncate(0)

		match := &entity.Match{
			ID:                 id,
			CreatedAt:          createdAt,
			RecordId:           recordId,
			DeckId:             deckId,
			UserId:             "",
			OpponentsUserId:    "",
			BO3Flg:             false,
			QualifyingRoundFlg: false,
			FinalTournamentFlg: false,
			DefaultVictoryFlg:  false,
			DefaultDefeatFlg:   false,
			VictoryFlg:         false,
			OpponentsDeckInfo:  "",
			Memo:               "",
		}

		var games []*usecase.GameParam
		games = append(
			games,
			usecase.NewGameParam(
				false,
				false,
				0,
				0,
				"",
			),
		)

		param := usecase.NewMatchParam(
			recordId,
			deckId,
			"",
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(match, nil)

		gameRequest := []*dto.GameRequest{
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		data := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId: recordId,
				DeckId:   deckId,
				Games:    gameRequest,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/matches", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		createdAt := time.Now().UTC().Truncate(0)

		match := &entity.Match{
			ID:                 id,
			CreatedAt:          createdAt,
			RecordId:           recordId,
			DeckId:             deckId,
			UserId:             uid,
			OpponentsUserId:    "",
			BO3Flg:             false,
			QualifyingRoundFlg: false,
			FinalTournamentFlg: false,
			DefaultVictoryFlg:  false,
			DefaultDefeatFlg:   false,
			VictoryFlg:         false,
			OpponentsDeckInfo:  "",
			Memo:               "",
		}

		var games []*usecase.GameParam
		games = append(
			games,
			usecase.NewGameParam(
				false,
				false,
				0,
				0,
				"",
			),
		)

		param := usecase.NewMatchParam(
			recordId,
			deckId,
			uid,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
		)

		mockUsecase.EXPECT().Create(context.Background(), param).Return(match, nil)

		gameRequest := []*dto.GameRequest{
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		data := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId: recordId,
				DeckId:   deckId,
				Games:    gameRequest,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/matches", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		recordId, _ := generateId()
		deckId := ""

		gameRequest := []*dto.GameRequest{
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		data := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId: recordId,
				DeckId:   deckId,
				Games:    gameRequest,
			},
		}

		mockUsecase.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, errors.New(""))

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/matches", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_MatchController_Update(t *testing.T) {
	t.Run("正常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		createdAt := time.Now().UTC().Truncate(0)

		match := &entity.Match{
			ID:                 id,
			CreatedAt:          createdAt,
			RecordId:           recordId,
			DeckId:             deckId,
			UserId:             "",
			OpponentsUserId:    "",
			BO3Flg:             false,
			QualifyingRoundFlg: false,
			FinalTournamentFlg: false,
			DefaultVictoryFlg:  false,
			DefaultDefeatFlg:   false,
			VictoryFlg:         false,
			OpponentsDeckInfo:  "",
			Memo:               "",
		}

		var games []*usecase.GameParam
		games = append(
			games,
			usecase.NewGameParam(
				false,
				false,
				0,
				0,
				"",
			),
		)

		param := usecase.NewMatchParam(
			recordId,
			deckId,
			"",
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
		)

		mockUsecase.EXPECT().Update(context.Background(), id, param).Return(match, nil)

		gameRequest := []*dto.GameRequest{
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		data := dto.MatchUpdateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId: recordId,
				DeckId:   deckId,
				Games:    gameRequest,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/matches/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにuidをセット
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		r.Use(func(ctx *gin.Context) {
			helper.SetUID(ctx, uid)
		})

		c, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		createdAt := time.Now().UTC().Truncate(0)

		match := &entity.Match{
			ID:                 id,
			CreatedAt:          createdAt,
			RecordId:           recordId,
			DeckId:             deckId,
			UserId:             uid,
			OpponentsUserId:    "",
			BO3Flg:             false,
			QualifyingRoundFlg: false,
			FinalTournamentFlg: false,
			DefaultVictoryFlg:  false,
			DefaultDefeatFlg:   false,
			VictoryFlg:         false,
			OpponentsDeckInfo:  "",
			Memo:               "",
		}

		var games []*usecase.GameParam
		games = append(
			games,
			usecase.NewGameParam(
				false,
				false,
				0,
				0,
				"",
			),
		)

		param := usecase.NewMatchParam(
			recordId,
			deckId,
			uid,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
		)

		mockUsecase.EXPECT().Update(context.Background(), id, param).Return(match, nil)

		gameRequest := []*dto.GameRequest{
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		data := dto.MatchUpdateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId: recordId,
				DeckId:   deckId,
				Games:    gameRequest,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/matches/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		r := gin.Default()
		c, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		gameRequest := []*dto.GameRequest{
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		data := dto.MatchUpdateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId: recordId,
				DeckId:   deckId,
				Games:    gameRequest,
			},
		}

		mockUsecase.EXPECT().Update(context.Background(), id, gomock.Any()).Return(nil, errors.New(""))

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/matches/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_MatchController_Delete(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestMatchController(t, r)

	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
