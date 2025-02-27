package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"go.uber.org/mock/gomock"
)

func TestMatchUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockMatchInterface(mockCtrl)
	usecase := NewMatch(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockMatchInterface,
		usecase MatchInterface,
	){
		"FindById":      test_MatchUsecase_FindById,
		"FindByMatchId": test_MatchUsecase_FindByRecordId,
		"Create":        test_MatchUsecase_Create,
		"Update":        test_MatchUsecase_Update,
		"Delete":        test_MatchUsecase_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_MatchUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockMatchInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		match := &entity.Match{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_MatchUsecase_FindByRecordId(t *testing.T, mockRepository *mock_repository.MockMatchInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		recordId, err := generateId()
		require.NoError(t, err)

		match := &entity.Match{
			ID:       id,
			RecordId: recordId,
		}

		matches := []*entity.Match{
			match,
		}

		mockRepository.EXPECT().FindByRecordId(context.Background(), recordId).Return(matches, nil)

		ret, err := usecase.FindByRecordId(context.Background(), recordId)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, recordId, ret[0].RecordId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		matchId, err := generateId()
		require.NoError(t, err)

		matches := []*entity.Match{}

		mockRepository.EXPECT().FindByRecordId(context.Background(), matchId).Return(matches, nil)

		ret, err := usecase.FindByRecordId(context.Background(), matchId)

		require.NoError(t, err)
		require.Equal(t, len(matches), len(ret))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		recordId, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindByRecordId(context.Background(), recordId).Return(nil, errors.New(""))

		ret, err := usecase.FindByRecordId(context.Background(), recordId)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_MatchUsecase_Create(t *testing.T, mockRepository *mock_repository.MockMatchInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				false,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			gameParams,
		)

		mockRepository.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), matchParam)

		require.NoError(t, err)

		require.IsType(t, string(""), ret.ID)
		require.IsType(t, time.Time{}, ret.CreatedAt)
		require.Equal(t, recordId, ret.RecordId)
		require.Equal(t, deckId, ret.DeckId)
		require.Equal(t, userId, ret.UserId)
		require.Equal(t, matchParam.OpponentsUserId, ret.OpponentsUserId)
		require.Equal(t, matchParam.BO3Flg, ret.BO3Flg)
		require.Equal(t, matchParam.QualifyingRoundFlg, ret.QualifyingRoundFlg)
		require.Equal(t, matchParam.FinalTournamentFlg, ret.FinalTournamentFlg)
		require.Equal(t, matchParam.DefaultVictoryFlg, ret.DefaultVictoryFlg)
		require.Equal(t, matchParam.DefaultDefeatFlg, ret.DefaultDefeatFlg)
		require.Equal(t, matchParam.VictoryFlg, ret.VictoryFlg)
		require.Equal(t, matchParam.OpponentsDeckInfo, ret.OpponentsDeckInfo)
		require.Equal(t, matchParam.Memo, ret.Memo)

		require.Equal(t, len(gameParams), len(ret.Games))
		require.IsType(t, string(""), ret.Games[0].ID)
		require.IsType(t, time.Time{}, ret.Games[0].CreatedAt)
		require.IsType(t, string(""), ret.Games[0].MatchId)
		require.Equal(t, userId, ret.Games[0].UserId)
		require.Equal(t, gameParams[0].GoFirst, ret.Games[0].GoFirst)
		require.Equal(t, gameParams[0].WinningFlg, ret.Games[0].WinningFlg)
		require.Equal(t, gameParams[0].YourPrizeCards, ret.Games[0].YourPrizeCards)
		require.Equal(t, gameParams[0].OpponentsPrizeCards, ret.Games[0].OpponentsPrizeCards)
		require.Equal(t, gameParams[0].Memo, ret.Games[0].Memo)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				true,
				0,
				0,
				"",
			),
			NewGameParam(
				false,
				true,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			gameParams,
		)

		mockRepository.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), matchParam)

		require.NoError(t, err)

		require.IsType(t, string(""), ret.ID)
		require.IsType(t, time.Time{}, ret.CreatedAt)
		require.Equal(t, recordId, ret.RecordId)
		require.Equal(t, deckId, ret.DeckId)
		require.Equal(t, userId, ret.UserId)
		require.Equal(t, matchParam.OpponentsUserId, ret.OpponentsUserId)
		require.Equal(t, matchParam.BO3Flg, ret.BO3Flg)
		require.Equal(t, matchParam.QualifyingRoundFlg, ret.QualifyingRoundFlg)
		require.Equal(t, matchParam.FinalTournamentFlg, ret.FinalTournamentFlg)
		require.Equal(t, matchParam.DefaultVictoryFlg, ret.DefaultVictoryFlg)
		require.Equal(t, matchParam.DefaultDefeatFlg, ret.DefaultDefeatFlg)
		require.Equal(t, matchParam.VictoryFlg, ret.VictoryFlg)
		require.Equal(t, matchParam.OpponentsDeckInfo, ret.OpponentsDeckInfo)
		require.Equal(t, matchParam.Memo, ret.Memo)

		require.Equal(t, len(gameParams), len(ret.Games))
		require.IsType(t, string(""), ret.Games[0].ID)
		require.IsType(t, time.Time{}, ret.Games[0].CreatedAt)
		require.IsType(t, string(""), ret.Games[0].MatchId)
		require.Equal(t, userId, ret.Games[0].UserId)
		require.Equal(t, gameParams[0].GoFirst, ret.Games[0].GoFirst)
		require.Equal(t, gameParams[0].WinningFlg, ret.Games[0].WinningFlg)
		require.Equal(t, gameParams[0].YourPrizeCards, ret.Games[0].YourPrizeCards)
		require.Equal(t, gameParams[0].OpponentsPrizeCards, ret.Games[0].OpponentsPrizeCards)
		require.Equal(t, gameParams[0].Memo, ret.Games[0].Memo)
		require.IsType(t, string(""), ret.Games[1].ID)
		require.IsType(t, time.Time{}, ret.Games[1].CreatedAt)
		require.IsType(t, string(""), ret.Games[1].MatchId)
		require.Equal(t, userId, ret.Games[1].UserId)
		require.Equal(t, gameParams[1].GoFirst, ret.Games[1].GoFirst)
		require.Equal(t, gameParams[1].WinningFlg, ret.Games[1].WinningFlg)
		require.Equal(t, gameParams[1].YourPrizeCards, ret.Games[1].YourPrizeCards)
		require.Equal(t, gameParams[1].OpponentsPrizeCards, ret.Games[1].OpponentsPrizeCards)
		require.Equal(t, gameParams[1].Memo, ret.Games[1].Memo)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				false,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			gameParams,
		)

		mockRepository.EXPECT().Create(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Create(context.Background(), matchParam)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_MatchUsecase_Update(t *testing.T, mockRepository *mock_repository.MockMatchInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		matchId, _ := generateId()
		datetime := time.Now().Truncate(0)
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		gameId, _ := generateId()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId,
				datetime,
				matchId,
				userId,
				true,
				true,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			games,
		)

		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(match, nil)
		mockRepository.EXPECT().Update(context.Background(), gomock.Any()).Return(nil)

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				false,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			gameParams,
		)

		ret, err := usecase.Update(context.Background(), matchId, matchParam)

		require.NoError(t, err)
		require.Equal(t, matchId, ret.ID)
		require.Equal(t, datetime, ret.CreatedAt)
		require.Equal(t, recordId, ret.RecordId)
		require.Equal(t, userId, ret.UserId)
		require.Equal(t, false, ret.VictoryFlg)

		require.Equal(t, len(gameParams), len(ret.Games))
		require.Equal(t, gameId, ret.Games[0].ID)
		require.Equal(t, true, ret.Games[0].GoFirst)
		require.Equal(t, false, ret.Games[0].WinningFlg)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		matchId, _ := generateId()
		datetime := time.Now().Truncate(0)
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		gameId, _ := generateId()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId,
				datetime,
				matchId,
				userId,
				true,
				true,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			games,
		)

		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(match, nil)
		mockRepository.EXPECT().Update(context.Background(), gomock.Any()).Return(nil)

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				false,
				0,
				0,
				"",
			),
			NewGameParam(
				true,
				true,
				0,
				0,
				"",
			),
			NewGameParam(
				false,
				true,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			gameParams,
		)

		ret, err := usecase.Update(context.Background(), matchId, matchParam)

		require.NoError(t, err)
		require.Equal(t, matchId, ret.ID)
		require.Equal(t, datetime, ret.CreatedAt)
		require.Equal(t, recordId, ret.RecordId)
		require.Equal(t, userId, ret.UserId)
		require.Equal(t, true, ret.BO3Flg)
		require.Equal(t, true, ret.VictoryFlg)

		require.Equal(t, len(gameParams), len(ret.Games))

		require.Equal(t, gameId, ret.Games[0].ID)
		require.Equal(t, true, ret.Games[0].GoFirst)
		require.Equal(t, false, ret.Games[0].WinningFlg)

		require.NotEmpty(t, ret.Games[1].ID)
		require.Equal(t, true, ret.Games[1].GoFirst)
		require.Equal(t, true, ret.Games[1].WinningFlg)

		require.NotEmpty(t, ret.Games[2].ID)
		require.Equal(t, false, ret.Games[2].GoFirst)
		require.Equal(t, true, ret.Games[2].WinningFlg)
	})

	t.Run("正常系_#03", func(t *testing.T) {
		matchId, _ := generateId()
		datetime := time.Now().Truncate(0)
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		gameId1, _ := generateId()
		gameId2, _ := generateId()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId1,
				datetime,
				matchId,
				userId,
				true,
				true,
				0,
				0,
				"",
			),
			entity.NewGame(
				gameId2,
				datetime,
				matchId,
				userId,
				true,
				true,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			games,
		)

		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(match, nil)
		mockRepository.EXPECT().Update(context.Background(), gomock.Any()).Return(nil)

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				false,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			gameParams,
		)

		ret, err := usecase.Update(context.Background(), matchId, matchParam)

		require.NoError(t, err)
		require.Equal(t, matchId, ret.ID)
		require.Equal(t, datetime, ret.CreatedAt)
		require.Equal(t, recordId, ret.RecordId)
		require.Equal(t, userId, ret.UserId)
		require.Equal(t, false, ret.BO3Flg)
		require.Equal(t, false, ret.VictoryFlg)

		require.Equal(t, len(gameParams), len(ret.Games))

		require.Equal(t, gameId1, ret.Games[0].ID)
		require.Equal(t, true, ret.Games[0].GoFirst)
		require.Equal(t, false, ret.Games[0].WinningFlg)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		matchId, _ := generateId()
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				false,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			gameParams,
		)

		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(nil, errors.New(""))

		ret, err := usecase.Update(context.Background(), matchId, matchParam)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		matchId, _ := generateId()
		datetime := time.Now().Truncate(0)
		recordId := "01JMPK4VF04QX714CG4PHYJ88K"
		deckId := "01JMKRNBW5TVN902YAE8GYZ367"
		userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		gameId, _ := generateId()

		var games []*entity.Game
		games = append(
			games,
			entity.NewGame(
				gameId,
				datetime,
				matchId,
				userId,
				true,
				true,
				0,
				0,
				"",
			),
		)

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			true,
			"",
			"",
			games,
		)

		var gameParams []*GameParam
		gameParams = append(
			gameParams,
			NewGameParam(
				true,
				false,
				0,
				0,
				"",
			),
		)

		matchParam := NewMatchParam(
			recordId,
			deckId,
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			"",
			"",
			gameParams,
		)

		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(match, nil)
		mockRepository.EXPECT().Update(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Update(context.Background(), matchId, matchParam)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_MatchUsecase_Delete(t *testing.T, mockRepository *mock_repository.MockMatchInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		err := usecase.Delete(context.Background(), id)

		require.NoError(t, err)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})
}
