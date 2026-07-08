package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// orderTrackingXxxEvaluation は、通知の作成順序(呼び出し順)を記録するためだけの
// 手書きスタブ(mock_usecaseはusecaseパッケージに依存しており、usecaseパッケージ内の
// テストから直接importするとimport cycleになるため、stubBadgeEvaluation等と同じ理由で
// gomockではなくこの手書きスタブを使う)。
type orderTrackingDesignationEvaluation struct {
	calls *[]string
}

func (s orderTrackingDesignationEvaluation) CurrentTier(ctx context.Context, userId string) (int, error) {
	return 0, nil
}

func (s orderTrackingDesignationEvaluation) TierAsOf(ctx context.Context, userId string, asOf time.Time) (int, error) {
	return 0, nil
}

func (s orderTrackingDesignationEvaluation) NotifyIfTierChanged(ctx context.Context, userId string, beforeTier int, achievedAt time.Time) {
	*s.calls = append(*s.calls, "designation")
}

func (s orderTrackingDesignationEvaluation) NotifyIfTierLost(ctx context.Context, userId string, beforeTier int) {
}

type orderTrackingBadgeEvaluation struct {
	calls *[]string
}

func (s orderTrackingBadgeEvaluation) EvaluateOnRecordCreated(ctx context.Context, userId string, record *entity.Record) ([]*entity.UserBadge, error) {
	return nil, nil
}

func (s orderTrackingBadgeEvaluation) EvaluateOnMatchCreated(ctx context.Context, userId string, match *entity.Match) ([]*entity.UserBadge, error) {
	*s.calls = append(*s.calls, "badge")
	return nil, nil
}

func (s orderTrackingBadgeEvaluation) EvaluateOnDeckCreated(ctx context.Context, userId string, deck *entity.Deck) ([]*entity.UserBadge, error) {
	return nil, nil
}

func (s orderTrackingBadgeEvaluation) EvaluateOnDeckCodeCreated(ctx context.Context, userId string, deckCode *entity.DeckCode) {
}

func (s orderTrackingBadgeEvaluation) EvaluateOnUserCreated(ctx context.Context, userId string, createdAt time.Time) ([]*entity.UserBadge, error) {
	return nil, nil
}

func (s orderTrackingBadgeEvaluation) EvaluateOnRecordDeleted(ctx context.Context, userId string) error {
	return nil
}

type orderTrackingEnvironmentBadgeEvaluation struct {
	calls *[]string
}

func (s orderTrackingEnvironmentBadgeEvaluation) EvaluateOnMatchCreated(ctx context.Context, userId string, match *entity.Match, basisTime time.Time) (*entity.Environment, error) {
	*s.calls = append(*s.calls, "environment_badge")
	return nil, nil
}

func (s orderTrackingEnvironmentBadgeEvaluation) NotifyAchieved(ctx context.Context, userId string, env *entity.Environment, achievedAt time.Time, isRead bool) (string, error) {
	return "", nil
}

func (s orderTrackingEnvironmentBadgeEvaluation) UpdateAchievedNotification(ctx context.Context, notificationId string, env *entity.Environment, achievedAt time.Time) error {
	return nil
}

// 通知の作成順は「ユーザバッジ→環境バッジ→称号/ランクアップ」である必要がある。
// created_at DESC(同値時はid DESC)で表示されるため、この作成順により表示順は下から
// 「ユーザバッジ→環境バッジ→称号/ランクアップ」(=上から称号/ランクアップ→環境バッジ→
// ユーザバッジ)になる。この呼び出し順が崩れると通知一覧の並び順バグが再発するため、
// 明示的に固定する。
func TestMatchUsecase_Create_NotificationCreationOrder(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockMatchInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	var calls []string
	usecase := NewMatch(
		mockRepository,
		mockRecordRepository,
		orderTrackingBadgeEvaluation{calls: &calls},
		orderTrackingDesignationEvaluation{calls: &calls},
		orderTrackingEnvironmentBadgeEvaluation{calls: &calls},
	)

	recordId := "01JMPK4VF04QX714CG4PHYJ88K"
	deckId := "01JMKRNBW5TVN902YAE8GYZ367"
	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	var gameParams []*GameParam
	gameParams = append(gameParams, NewGameParam(true, false, 0, 0, ""))

	matchParam := NewMatchParam(
		recordId, deckId, "", userId, "",
		false, false, false, false, false, false, false, false,
		"", "", gameParams, nil,
	)

	mockRepository.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)
	// OfficialEventId != 0 の記録として扱い、環境バッジ評価も呼ばれる経路を通す。
	mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, OfficialEventId: 1}, nil)

	_, err := usecase.Create(context.Background(), matchParam)

	require.NoError(t, err)
	require.Equal(t, []string{"badge", "environment_badge", "designation"}, calls)
}

func TestMatchUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockMatchInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	usecase := NewMatch(mockRepository, mockRecordRepository, stubBadgeEvaluation{}, stubDesignationEvaluation{}, stubEnvironmentBadgeEvaluation{})

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockMatchInterface,
		mockRecordRepository *mock_repository.MockRecordInterface,
		usecase MatchInterface,
	){
		"FindById":      test_MatchUsecase_FindById,
		"FindByMatchId": test_MatchUsecase_FindByRecordId,
		"Create":        test_MatchUsecase_Create,
		"Update":        test_MatchUsecase_Update,
		"Delete":        test_MatchUsecase_Delete,
		"Reorder":       test_MatchUsecase_Reorder,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, mockRecordRepository, usecase)
		})
	}
}

func test_MatchUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
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

func test_MatchUsecase_FindByRecordId(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
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

func test_MatchUsecase_Create(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
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
			gameParams,
			pokemonSpriteParams,
		)

		mockRepository.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId}, nil)

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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			false,
			true,
			false,
			"",
			"",
			gameParams,
			pokemonSpriteParams,
		)

		mockRepository.EXPECT().Create(context.Background(), gomock.Any()).Return(nil)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId}, nil)

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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
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
			gameParams,
			pokemonSpriteParams,
		)

		mockRepository.EXPECT().Create(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Create(context.Background(), matchParam)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_MatchUsecase_Update(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		matchId, _ := generateId()
		datetime := time.Now().Local()
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

		var pokemonSprite []*entity.PokemonSprite

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			"",
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			true,
			false,
			"",
			"",
			games,
			pokemonSprite,
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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
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
			gameParams,
			pokemonSpriteParams,
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
		datetime := time.Now().Local()
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

		var pokemonSprite []*entity.PokemonSprite

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			"",
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			true,
			false,
			"",
			"",
			games,
			pokemonSprite,
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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			false,
			true,
			false,
			"",
			"",
			gameParams,
			pokemonSpriteParams,
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
		datetime := time.Now().Local()
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

		var pokemonSprite []*entity.PokemonSprite

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			"",
			userId,
			"",
			true,
			false,
			false,
			false,
			false,
			false,
			true,
			false,
			"",
			"",
			games,
			pokemonSprite,
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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
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
			gameParams,
			pokemonSpriteParams,
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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
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
			gameParams,
			pokemonSpriteParams,
		)

		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(nil, errors.New(""))

		ret, err := usecase.Update(context.Background(), matchId, matchParam)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		matchId, _ := generateId()
		datetime := time.Now().Local()
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

		var pokemonSprite []*entity.PokemonSprite

		match := entity.NewMatch(
			matchId,
			datetime,
			recordId,
			deckId,
			"",
			userId,
			"",
			false,
			false,
			false,
			false,
			false,
			false,
			true,
			false,
			"",
			"",
			games,
			pokemonSprite,
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

		var pokemonSpriteParams []*PokemonSpriteParam

		matchParam := NewMatchParam(
			recordId,
			deckId,
			"",
			userId,
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
			gameParams,
			pokemonSpriteParams,
		)

		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(match, nil)
		mockRepository.EXPECT().Update(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Update(context.Background(), matchId, matchParam)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_MatchUsecase_Delete(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		match := &entity.Match{ID: id, UserId: "user-1"}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		err := usecase.Delete(context.Background(), id)

		require.NoError(t, err)
	})

	t.Run("異常系_#01_FindByIdが失敗する場合", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_#02_Deleteが失敗する場合", func(t *testing.T) {
		id, _ := generateId()
		match := &entity.Match{ID: id, UserId: "user-1"}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(match, nil)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})
}

func test_MatchUsecase_Reorder(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		recordId, _ := generateId()
		id1, _ := generateId()
		id2, _ := generateId()

		orders := []*entity.MatchOrder{
			{ID: id1, QualifyingRoundFlg: false, FinalTournamentFlg: true},
			{ID: id2, QualifyingRoundFlg: true, FinalTournamentFlg: false},
		}

		mockRepository.EXPECT().Reorder(context.Background(), recordId, orders).Return(nil)

		err := usecase.Reorder(context.Background(), recordId, orders)

		require.NoError(t, err)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		recordId, _ := generateId()
		orders := []*entity.MatchOrder{}

		mockRepository.EXPECT().Reorder(context.Background(), recordId, orders).Return(errors.New(""))

		err := usecase.Reorder(context.Background(), recordId, orders)

		require.Equal(t, err, errors.New(""))
	})
}
