package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// BO3(2本先取)の対戦結果に対する Usecase 層の振る舞いを検証する。
// 特に Update 時の「既存Gameの引き継ぎ」と「ゲーム数の増減」を確認する。
func TestMatchUsecaseBO3(t *testing.T) {
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
		"Create_BO3は3ゲームがそのまま永続化される":     test_MatchUsecaseBO3_Create,
		"Update_3ゲームから2ゲームへ減らせる":        test_MatchUsecaseBO3_UpdateShrink,
		"Update_1ゲームから3ゲームへ増やせる":        test_MatchUsecaseBO3_UpdateGrow,
		"Update_既存GameのIDとCreatedAtを引き継ぐ": test_MatchUsecaseBO3_UpdateKeepsIdentity,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, mockRecordRepository, usecase)
		})
	}
}

// bo3GameParams はBO3のゲーム入力(勝敗の並び)を GameParam に変換する。
func bo3GameParams(winningFlgs ...bool) []*GameParam {
	var params []*GameParam
	for i, w := range winningFlgs {
		// 先攻/後攻は 1本目:先攻, 2本目:後攻, 3本目:先攻 を想定
		params = append(params, NewGameParam(i%2 == 0, w, 0, 0, ""))
	}
	return params
}

// bo3MatchParam はBO3の対戦結果パラメータを組み立てる。
// victoryFlg は「2本先取できたか」= 勝ちゲーム数が2以上か。
func bo3MatchParam(recordId string, userId string, games []*GameParam) *MatchParam {
	wins := 0
	for _, g := range games {
		if g.WinningFlg {
			wins++
		}
	}

	return NewMatchParam(
		recordId,
		"",
		"",
		userId,
		"",
		true, // bo3Flg
		false,
		false,
		false,
		false,
		false,
		wins >= 2, // victoryFlg: 先取2本
		false,
		"リザードンex",
		"",
		games,
		nil,
	)
}

// 既存のBO3対戦(gameCount本のゲームを持つ)を表すエンティティを組み立てる。
func existingBO3Match(t *testing.T, id string, recordId string, userId string, gameCount int) *entity.Match {
	t.Helper()

	var games []*entity.Game
	for i := range gameCount {
		gameId, err := generateId()
		require.NoError(t, err)

		games = append(games, entity.NewGame(
			gameId,
			time.Now().Local().Add(time.Duration(i)*time.Second),
			id,
			userId,
			true,
			true,
			0,
			0,
			"",
		))
	}

	return entity.NewMatch(
		id, time.Now().Local(), recordId, "", "", userId, "",
		true, false, false, false, false, false, true, false,
		"リザードンex", "", games, nil,
	)
}

// BO3で3ゲーム(2-1)を作成すると、3ゲームがそのまま永続化されること。
func test_MatchUsecaseBO3_Create(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
	t.Run("正常系_3ゲーム(2-1で勝利)", func(t *testing.T) {
		ctx := context.Background()
		recordId, err := generateId()
		require.NoError(t, err)
		userId := "bo3-user"

		// 1本目:勝ち, 2本目:負け, 3本目:勝ち → 2-1で勝利
		param := bo3MatchParam(recordId, userId, bo3GameParams(true, false, true))

		var saved *entity.Match
		mockRepository.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, m *entity.Match) error {
				saved = m
				return nil
			},
		)
		// 環境バッジ判定用に親recordを引くが、公式イベント以外(officialEventId=0)なら判定しない
		mockRecordRepository.EXPECT().FindById(ctx, recordId).Return(
			entity.NewRecord(recordId, time.Now(), 0, "", "", "", userId, "", "", time.Now(), false, false, "", ""),
			nil,
		).AnyTimes()

		match, err := usecase.Create(ctx, param)
		require.NoError(t, err)

		require.True(t, match.BO3Flg, "bo3_flg が true であること")
		require.True(t, match.VictoryFlg, "2-1なので勝利")
		require.Len(t, saved.Games, 3, "3ゲームが永続化されること")

		// 各ゲームに一意なIDが採番され、matchIdが紐づいていること
		ids := map[string]bool{}
		for _, g := range saved.Games {
			require.NotEmpty(t, g.ID)
			require.False(t, ids[g.ID], "ゲームIDが重複してはならない")
			ids[g.ID] = true
			require.Equal(t, match.ID, g.MatchId)
		}

		// 勝敗の並びが入力どおりであること
		require.True(t, saved.Games[0].WinningFlg)
		require.False(t, saved.Games[1].WinningFlg)
		require.True(t, saved.Games[2].WinningFlg)
	})
}

// 3ゲームで登録したBO3を「実は2-0だった」と2ゲームに修正できること。
func test_MatchUsecaseBO3_UpdateShrink(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
	t.Run("正常系_3ゲーム→2ゲーム", func(t *testing.T) {
		ctx := context.Background()
		matchId, err := generateId()
		require.NoError(t, err)
		recordId, err := generateId()
		require.NoError(t, err)
		userId := "bo3-user"

		existing := existingBO3Match(t, matchId, recordId, userId, 3)
		mockRepository.EXPECT().FindById(ctx, matchId).Return(existing, nil)

		var saved *entity.Match
		mockRepository.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, m *entity.Match) error {
				saved = m
				return nil
			},
		)

		// 2-0(2ゲーム)に修正
		param := bo3MatchParam(recordId, userId, bo3GameParams(true, true))

		match, err := usecase.Update(ctx, matchId, param)
		require.NoError(t, err)

		require.Len(t, saved.Games, 2, "2ゲームに減っていること")
		require.True(t, match.VictoryFlg, "2-0なので勝利")

		// 残る2ゲームは既存GameのIDを引き継いでいること(新規採番されない)
		require.Equal(t, existing.Games[0].ID, saved.Games[0].ID)
		require.Equal(t, existing.Games[1].ID, saved.Games[1].ID)
	})
}

// BO1(1ゲーム)で登録した対戦をBO3(3ゲーム)に修正できること。
func test_MatchUsecaseBO3_UpdateGrow(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
	t.Run("正常系_1ゲーム→3ゲーム", func(t *testing.T) {
		ctx := context.Background()
		matchId, err := generateId()
		require.NoError(t, err)
		recordId, err := generateId()
		require.NoError(t, err)
		userId := "bo3-user"

		existing := existingBO3Match(t, matchId, recordId, userId, 1)
		mockRepository.EXPECT().FindById(ctx, matchId).Return(existing, nil)

		var saved *entity.Match
		mockRepository.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, m *entity.Match) error {
				saved = m
				return nil
			},
		)

		// 1本目:勝ち, 2本目:負け, 3本目:勝ち
		param := bo3MatchParam(recordId, userId, bo3GameParams(true, false, true))

		_, err = usecase.Update(ctx, matchId, param)
		require.NoError(t, err)

		require.Len(t, saved.Games, 3, "3ゲームに増えていること")

		// 1本目は既存Gameを引き継ぎ、2/3本目は新規採番されること
		require.Equal(t, existing.Games[0].ID, saved.Games[0].ID)
		require.NotEmpty(t, saved.Games[1].ID)
		require.NotEmpty(t, saved.Games[2].ID)
		require.NotEqual(t, saved.Games[0].ID, saved.Games[1].ID)
		require.NotEqual(t, saved.Games[1].ID, saved.Games[2].ID)

		// 勝敗の並びが入力どおりであること
		require.True(t, saved.Games[0].WinningFlg)
		require.False(t, saved.Games[1].WinningFlg)
		require.True(t, saved.Games[2].WinningFlg)
	})
}

// 既存ゲームを上書きする場合、ID と CreatedAt が引き継がれること
// (created_at が変わると ORDER BY games.created_at ASC の並び順が壊れるため)。
func test_MatchUsecaseBO3_UpdateKeepsIdentity(t *testing.T, mockRepository *mock_repository.MockMatchInterface, mockRecordRepository *mock_repository.MockRecordInterface, usecase MatchInterface) {
	t.Run("正常系_ID/CreatedAtの引き継ぎ", func(t *testing.T) {
		ctx := context.Background()
		matchId, err := generateId()
		require.NoError(t, err)
		recordId, err := generateId()
		require.NoError(t, err)
		userId := "bo3-user"

		existing := existingBO3Match(t, matchId, recordId, userId, 3)
		mockRepository.EXPECT().FindById(ctx, matchId).Return(existing, nil)

		var saved *entity.Match
		mockRepository.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, m *entity.Match) error {
				saved = m
				return nil
			},
		)

		// 3ゲームのまま、勝敗だけを 1勝2敗(敗北)に修正
		param := bo3MatchParam(recordId, userId, bo3GameParams(false, true, false))

		match, err := usecase.Update(ctx, matchId, param)
		require.NoError(t, err)

		require.Len(t, saved.Games, 3)
		require.False(t, match.VictoryFlg, "1勝2敗なので敗北")

		for i := range 3 {
			require.Equal(t, existing.Games[i].ID, saved.Games[i].ID, "%d本目のIDが引き継がれること", i+1)
			require.Equal(t, existing.Games[i].CreatedAt, saved.Games[i].CreatedAt, "%d本目のCreatedAtが引き継がれること", i+1)
		}

		// 勝敗が上書きされていること
		require.False(t, saved.Games[0].WinningFlg)
		require.True(t, saved.Games[1].WinningFlg)
		require.False(t, saved.Games[2].WinningFlg)
	})
}
