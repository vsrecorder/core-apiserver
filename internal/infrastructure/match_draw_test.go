package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// 両者引き分け(ダブルドロー)のBO3を実DBに対して検証する。
// 引き分けは1勝1敗の2ゲームで決着し、victory_flg=false / draw_flg=true になる。
//
// 実行方法:
//
//	TEST_DB_DSN="host=localhost port=55433 user=vsr password=vsr dbname=vsrtest sslmode=disable" go test ./internal/infrastructure/ -run TestMatchBO3Draw -v

// newBO3DrawMatch は両者引き分けのBO3(1勝1敗の2ゲーム)を組み立てる。
func newBO3DrawMatch(matchId string, games []*entity.Game) *entity.Match {
	return entity.NewMatch(
		matchId, time.Now().Local(), bo3TestRecordId, "", "", bo3TestUserId, "",
		true,  // bo3Flg
		false, // groupMatchFlg
		false, false, false, false,
		false, // victoryFlg（引き分けは勝ちでない）
		true,  // drawFlg
		false,
		"リザードンex", "", games, nil,
	)
}

func TestMatchBO3Draw(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"Create_1勝1敗の両者引き分けを作成して取得できる": test_MatchBO3Draw_Create,
		"Update_勝ち(2-1)から引き分け(1-1)へ変更できる": test_MatchBO3Draw_UpdateWinToDraw,
		"Update_引き分け(1-1)から勝ち(2-1)へ変更できる": test_MatchBO3Draw_UpdateDrawToWin,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// 1勝1敗の両者引き分けを作成し、draw_flg=true / victory_flg=false / 2ゲームで取得できること。
func test_MatchBO3Draw_Create(t *testing.T) {
	r, _ := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3DRAW0000000000001"
	games := []*entity.Game{
		newBO3Game(matchId, 1, true, true),   // 1本目: 勝ち
		newBO3Game(matchId, 2, false, false), // 2本目: 負け
	}
	require.NoError(t, r.Create(ctx, newBO3DrawMatch(matchId, games)))

	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)

	require.True(t, got.BO3Flg)
	require.True(t, got.DrawFlg, "両者引き分けなので draw_flg は true")
	require.False(t, got.VictoryFlg, "引き分けは勝ちではない")
	require.Len(t, got.Games, 2, "引き分けは1勝1敗の2ゲーム")
	require.Equal(t, entity.MatchResultDraw, got.Result())
}

// 勝ち(2-1, 3ゲーム)を両者引き分け(1-1, 2ゲーム)へ更新すると、
// 3本目が削除され draw_flg=true / victory_flg=false になること。
func test_MatchBO3Draw_UpdateWinToDraw(t *testing.T) {
	r, db := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3DRAW0000000000002"
	// 2-1で勝利(3ゲーム)
	require.NoError(t, r.Create(ctx, newBO3Match(matchId, true, []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, false),
		newBO3Game(matchId, 3, true, true),
	})))

	// 「実は1勝1敗で時間切れ引き分けだった」と2ゲームへ修正
	require.NoError(t, r.Update(ctx, newBO3DrawMatch(matchId, []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, false),
	})))

	// 3本目のgamesは論理削除されている
	var deletedCount int64
	require.NoError(t, db.Raw(
		"SELECT COUNT(*) FROM games WHERE match_id = ? AND deleted_at IS NOT NULL", matchId,
	).Scan(&deletedCount).Error)
	require.Equal(t, int64(1), deletedCount, "3本目が論理削除されていること")

	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)
	require.True(t, got.DrawFlg, "draw_flg が true に更新されていること")
	require.False(t, got.VictoryFlg, "victory_flg が false に更新されていること")
	require.Len(t, got.Games, 2, "2ゲームに減っていること(論理削除済みは含まない)")
	require.Equal(t, entity.MatchResultDraw, got.Result())
}

// 両者引き分け(1-1, 2ゲーム)を勝ち(2-1, 3ゲーム)へ更新すると、
// 3本目が追加され draw_flg=false / victory_flg=true になること。
func test_MatchBO3Draw_UpdateDrawToWin(t *testing.T) {
	r, _ := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3DRAW0000000000003"
	// 1勝1敗の引き分け(2ゲーム)
	require.NoError(t, r.Create(ctx, newBO3DrawMatch(matchId, []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, false),
	})))

	// 「3本目を行って勝った」と2-1(3ゲーム)へ修正
	require.NoError(t, r.Update(ctx, newBO3Match(matchId, true, []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, false),
		newBO3Game(matchId, 3, true, true),
	})))

	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)
	require.False(t, got.DrawFlg, "draw_flg が false に更新されていること")
	require.True(t, got.VictoryFlg, "victory_flg が true に更新されていること")
	require.Len(t, got.Games, 3, "3ゲームに増えていること")
	require.Equal(t, "3本目", got.Games[2].Memo, "追加された3本目が末尾に並ぶこと")
	require.Equal(t, entity.MatchResultWin, got.Result())
}
