package validation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
)

func TestMatchValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"MatchCreateMiddleware": test_MatchCreateMiddleware,
		//"MatchUpdateMiddleware": test_MatchUpdateMiddleware,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_MatchCreateMiddleware(t *testing.T) {
	t.Run("正常系_BO1_1ゲーム勝利のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_BO1_1ゲーム敗北のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_BO3_2連勝で勝利のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_BO3_2連敗で敗北のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_BO3_勝敗勝で勝利のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_BO3_敗勝勝で勝利のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_BO3_勝敗敗で敗北のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_BO3_敗勝敗で敗北のリクエストを受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_不戦勝はゲームなしで受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := true
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("正常系_不戦敗はゲームなしで受理する", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := true
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		actual := helper.GetMatchCreateRequest(ginContext)
		require.Equal(t, expected, actual)
	})

	t.Run("異常系_recordIdが空なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := ""
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO1でゲームが2件なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO1でゲーム結果と勝敗フラグが矛盾したら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_2ゲーム1勝1敗で勝利指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_2ゲーム1勝1敗で敗北指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_2連敗なのに勝利指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_2連勝なのに敗北指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3でゲームが1件なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3でゲームが4件なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_2連勝で決着後に3ゲーム目があれば400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_2連敗で決着後に3ゲーム目があれば400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_勝敗勝なのに敗北指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_敗勝勝なのに敗北指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_勝敗敗なのに勝利指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_BO3_敗勝敗なのに勝利指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
			{
				GoFirst:             false,
				WinningFlg:          false,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := true
		defaultVictoryFlg := false
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_不戦勝なのに敗北指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := true
		defaultDefeatFlg := false
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_不戦敗なのに勝利指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := true
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_不戦勝と不戦敗の同時指定なら400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := true
		defaultDefeatFlg := true
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_不戦勝なのにゲームがあれば400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := true
		defaultDefeatFlg := false
		victoryFlg := true

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("異常系_不戦敗なのにゲームがあれば400を返す", func(t *testing.T) {
		w := httptest.NewRecorder()
		ginContext, _ := gin.CreateTestContext(w)

		games := []*dto.GameRequest{
			{
				GoFirst:             true,
				WinningFlg:          true,
				YourPrizeCards:      0,
				OpponentsPrizeCards: 0,
				Memo:                "",
			},
		}

		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		bo3Flg := false
		defaultVictoryFlg := false
		defaultDefeatFlg := true
		victoryFlg := false

		expected := dto.MatchCreateRequest{
			MatchRequest: dto.MatchRequest{
				RecordId:           recordId,
				DeckId:             deckId,
				OpponentsUserId:    "",
				BO3Flg:             bo3Flg,
				QualifyingRoundFlg: false,
				FinalTournamentFlg: false,
				DefaultVictoryFlg:  defaultVictoryFlg,
				DefaultDefeatFlg:   defaultDefeatFlg,
				VictoryFlg:         victoryFlg,
				OpponentsDeckInfo:  "",
				Memo:               "",
				Games:              games,
			},
		}

		dataBytes, err := json.Marshal(expected)
		require.NoError(t, err)

		// Middlewareのテストのためpathは何でもよい
		req, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
		require.NoError(t, err)

		ginContext.Request = req

		middleware := MatchCreateMiddleware()
		middleware(ginContext)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
