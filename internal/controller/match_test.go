package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func setupMock4TestMatchController(t *testing.T) (*mock_repository.MockMatchInterface, *mock_repository.MockRecordInterface, *mock_usecase.MockMatchInterface) {
	mockCtrl := gomock.NewController(t)
	mockMatchRepository := mock_repository.NewMockMatchInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	mockUsecase := mock_usecase.NewMockMatchInterface(mockCtrl)

	return mockMatchRepository, mockRecordRepository, mockUsecase
}

func setup4TestMatchController(t *testing.T, r *gin.Engine) (
	*Match,
	*mock_repository.MockMatchInterface,
	*mock_repository.MockRecordInterface,
	*mock_usecase.MockMatchInterface,
) {
	mockMatchRepository, mockRecordRepository, mockUsecase := setupMock4TestMatchController(t)

	c := NewMatch(r, mockMatchRepository, mockRecordRepository, mockUsecase)
	c.RegisterRoute("")

	return c, mockMatchRepository, mockRecordRepository, mockUsecase
}

func TestMatchController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetById":       test_MatchController_GetById,
		"GetByRecordId": test_MatchController_GetByRecordId,
		"Create":        test_MatchController_Create,
		"Update":        test_MatchController_Update,
		"Delete":        test_MatchController_Delete,
		"Reorder":       test_MatchController_Reorder,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_MatchController_GetById(t *testing.T) {
	t.Run("正常系_指定IDのマッチを返す", func(t *testing.T) {
		r := gin.Default()
		c, mockMatchRepository, mockRecordRepository, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()
		createdAt := time.Now().Local()
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

		// MatchGetByIdAuthorizationMiddlewareが参照する
		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid, PrivateFlg: false}, nil)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(match, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_マッチが存在しなければ404を返す", func(t *testing.T) {
		r := gin.Default()
		c, mockMatchRepository, mockRecordRepository, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()
		recordId, _ := generateId()
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		match := &entity.Match{
			ID:       id,
			RecordId: recordId,
			UserId:   uid,
		}

		// MatchGetByIdAuthorizationMiddlewareが参照する
		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid, PrivateFlg: false}, nil)

		mockUsecase.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/matches/"+id, nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res dto.MatchGetByIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()
		c, mockMatchRepository, mockRecordRepository, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()
		recordId, _ := generateId()
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		match := &entity.Match{
			ID:       id,
			RecordId: recordId,
			UserId:   uid,
		}

		// MatchGetByIdAuthorizationMiddlewareが参照する
		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid, PrivateFlg: false}, nil)

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
	t.Run("正常系_指定記録のマッチ一覧を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, mockRecordRepository, mockUsecase := setup4TestMatchController(t, r)

		id, _ := generateId()
		createdAt := time.Now().Local()
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

		// MatchGetByRecordIdAuthorizationMiddlewareが参照する
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid, PrivateFlg: false}, nil)

		mockUsecase.EXPECT().FindByRecordId(context.Background(), recordId).Return(matches, nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records/"+recordId+"/matches", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res []*dto.MatchResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res[0].ID)
		//require.Equal(t, createdAt, res[0].CreatedAt)
		require.Equal(t, recordId, res[0].RecordId)
		require.Equal(t, uid, res[0].UserId)
	})

	t.Run("正常系_該当なしの場合も200で空一覧を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, mockRecordRepository, mockUsecase := setup4TestMatchController(t, r)

		recordId, _ := generateId()
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// MatchGetByRecordIdAuthorizationMiddlewareが参照する
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid, PrivateFlg: false}, nil)

		mockUsecase.EXPECT().FindByRecordId(context.Background(), recordId).Return(nil, apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("GET", "/records/"+recordId+"/matches", nil)
		require.NoError(t, err)

		c.router.ServeHTTP(w, req)

		var res []*dto.MatchGetByRecordIdResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()
		c, _, mockRecordRepository, mockUsecase := setup4TestMatchController(t, r)

		recordId, _ := generateId()
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		// MatchGetByRecordIdAuthorizationMiddlewareが参照する
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid, PrivateFlg: false}, nil)

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

	t.Run("正常系_マッチを作成する", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにJWTを生成
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, _, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		createdAt := time.Now().Local()

		match := &entity.Match{
			ID:                 id,
			CreatedAt:          createdAt,
			RecordId:           recordId,
			DeckId:             deckId,
			DeckCodeId:         "",
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

		var pokemonSprites []*usecase.PokemonSpriteParam

		param := usecase.NewMatchParam(
			recordId,
			deckId,
			"",
			uid,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
			pokemonSprites,
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
				RecordId:   recordId,
				DeckId:     deckId,
				DeckCodeId: "",
				Games:      gameRequest,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("POST", "/matches", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.MatchCreateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, _, _, mockUsecase := setup4TestMatchController(t, r)

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
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_MatchController_Update(t *testing.T) {

	t.Run("正常系_本人のマッチを更新する", func(t *testing.T) {
		r := gin.Default()

		// 認証済みとするためにJWTを生成
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockMatchRepository, _, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		createdAt := time.Now().Local()

		match := &entity.Match{
			ID:                 id,
			CreatedAt:          createdAt,
			RecordId:           recordId,
			DeckId:             deckId,
			DeckCodeId:         "",
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

		// MatchUpdateAuthorizationMiddlewareが本人確認のために参照する
		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(&entity.Match{ID: id, UserId: uid}, nil)

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

		var pokemonSprites []*usecase.PokemonSpriteParam

		param := usecase.NewMatchParam(
			recordId,
			deckId,
			"",
			uid,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			games,
			pokemonSprites,
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
				RecordId:   recordId,
				DeckId:     deckId,
				DeckCodeId: "",
				Games:      gameRequest,
			},
		}

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/matches/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		var res dto.MatchUpdateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, id, res.ID)
		//require.Equal(t, createdAt, res.CreatedAt)
		require.Equal(t, recordId, res.RecordId)
		require.Equal(t, uid, res.UserId)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		r := gin.Default()

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		secretKey, err := testutil.GenerateJWTSecret()
		require.NoError(t, err)
		os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

		c, mockMatchRepository, _, mockUsecase := setup4TestMatchController(t, r)

		id, err := generateId()
		require.NoError(t, err)
		recordId, _ := generateId()
		deckId := ""

		// MatchUpdateAuthorizationMiddlewareが本人確認のために参照する
		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(&entity.Match{ID: id, UserId: uid}, nil)

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
				RecordId:   recordId,
				DeckId:     deckId,
				DeckCodeId: "",
				Games:      gameRequest,
			},
		}

		mockUsecase.EXPECT().Update(context.Background(), id, gomock.Any()).Return(nil, errors.New(""))

		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/matches/"+id, strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_MatchController_Delete(t *testing.T) {
	r := gin.Default()
	c, mockMatchRepository, _, mockUsecase := setup4TestMatchController(t, r)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	t.Run("正常系_本人のマッチを削除する", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		// MatchDeleteAuthorizationMiddlewareが本人確認のために参照する
		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(&entity.Match{ID: id, UserId: uid}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(nil)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/matches/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_削除対象が存在しなければ400を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(&entity.Match{ID: id, UserId: uid}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(apperror.ErrRecordNotFound)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/matches/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockMatchRepository.EXPECT().FindById(context.Background(), id).Return(&entity.Match{ID: id, UserId: uid}, nil)
		mockUsecase.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		w := httptest.NewRecorder()

		req, err := http.NewRequest("DELETE", "/matches/"+id, nil)
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func test_MatchController_Reorder(t *testing.T) {
	r := gin.Default()
	c, _, mockRecordRepository, mockUsecase := setup4TestMatchController(t, r)

	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	os.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	t.Run("正常系_本人の記録のマッチ並び替えを受け付ける", func(t *testing.T) {
		recordId, _ := generateId()
		id1, _ := generateId()
		id2, _ := generateId()

		// MatchReorderAuthorizationMiddlewareが本人確認のために参照する
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid}, nil)

		orders := []*entity.MatchOrder{
			{ID: id1, QualifyingRoundFlg: false, FinalTournamentFlg: true},
			{ID: id2, QualifyingRoundFlg: true, FinalTournamentFlg: false},
		}
		mockUsecase.EXPECT().Reorder(context.Background(), recordId, orders).Return(nil)

		data := dto.MatchReorderRequest{
			Matches: []*dto.MatchOrderItem{
				{Id: id1, QualifyingRoundFlg: false, FinalTournamentFlg: true},
				{Id: id2, QualifyingRoundFlg: true, FinalTournamentFlg: false},
			},
		}
		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/records/"+recordId+"/matches/order", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("異常系_他人のrecord", func(t *testing.T) {
		recordId, _ := generateId()
		id1, _ := generateId()

		// record所有者が別のユーザー
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: "other-user"}, nil)

		data := dto.MatchReorderRequest{
			Matches: []*dto.MatchOrderItem{
				{Id: id1, QualifyingRoundFlg: false, FinalTournamentFlg: false},
			},
		}
		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/records/"+recordId+"/matches/order", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_matchesが空", func(t *testing.T) {
		recordId, _ := generateId()

		// MatchReorderAuthorizationMiddlewareが本人確認のために参照する
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid}, nil)

		data := dto.MatchReorderRequest{
			Matches: []*dto.MatchOrderItem{},
		}
		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/records/"+recordId+"/matches/order", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_usecaseがErrInvalidMatchOrderを返す", func(t *testing.T) {
		recordId, _ := generateId()
		id1, _ := generateId()

		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid}, nil)

		orders := []*entity.MatchOrder{
			{ID: id1, QualifyingRoundFlg: false, FinalTournamentFlg: false},
		}
		mockUsecase.EXPECT().Reorder(context.Background(), recordId, orders).Return(apperror.ErrInvalidMatchOrder)

		data := dto.MatchReorderRequest{
			Matches: []*dto.MatchOrderItem{
				{Id: id1, QualifyingRoundFlg: false, FinalTournamentFlg: false},
			},
		}
		dataBytes, err := json.Marshal(data)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		req, err := http.NewRequest("PUT", "/records/"+recordId+"/matches/order", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)
		setJWTAuthHeader(t, req, uid, secretKey)

		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
