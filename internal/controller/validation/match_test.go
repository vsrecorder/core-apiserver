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
	t.Run("正常系_#01-1", func(t *testing.T) {
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

	t.Run("正常系_#01-2", func(t *testing.T) {
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

	t.Run("正常系_#02-1", func(t *testing.T) {
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

	t.Run("正常系_#02-2", func(t *testing.T) {
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

	t.Run("正常系_#03-1-1", func(t *testing.T) {
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

	t.Run("正常系_#03-1-2", func(t *testing.T) {
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

	t.Run("正常系_#03-2-1", func(t *testing.T) {
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

	t.Run("正常系_#03-2-2", func(t *testing.T) {
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

	t.Run("正常系_#04-1", func(t *testing.T) {
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

	t.Run("正常系_#04-2", func(t *testing.T) {
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

	t.Run("異常系_#00", func(t *testing.T) {
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

	t.Run("異常系_#01-1", func(t *testing.T) {
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

	t.Run("異常系_#01-2", func(t *testing.T) {
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

	t.Run("異常系_#02-1", func(t *testing.T) {
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

	t.Run("異常系_#02-2", func(t *testing.T) {
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

	t.Run("異常系_#02-3", func(t *testing.T) {
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

	t.Run("異常系_#02-4", func(t *testing.T) {
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

	t.Run("異常系_#03-1", func(t *testing.T) {
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

	t.Run("異常系_#03-2", func(t *testing.T) {
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

	t.Run("異常系_#03-3", func(t *testing.T) {
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

	t.Run("異常系_#03-4", func(t *testing.T) {
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

	t.Run("異常系_#03-5-1", func(t *testing.T) {
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

	t.Run("異常系_#03-5-2", func(t *testing.T) {
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

	t.Run("異常系_#03-6-1", func(t *testing.T) {
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

	t.Run("異常系_#03-6-2", func(t *testing.T) {
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

	t.Run("異常系_#04-1", func(t *testing.T) {
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

	t.Run("異常系_#04-2", func(t *testing.T) {
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

	t.Run("異常系_#04-3", func(t *testing.T) {
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

	t.Run("異常系_#04-4", func(t *testing.T) {
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

	t.Run("異常系_#04-5", func(t *testing.T) {
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
