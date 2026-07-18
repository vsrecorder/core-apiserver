package infrastructure

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// BO3(2本先取)の対戦結果を実DBに対して検証する統合テスト。
//
// 実行方法:
//
//	TEST_DB_DSN="host=localhost port=55433 user=vsr password=vsr dbname=vsrtest sslmode=disable" go test ./internal/infrastructure/ -run TestMatchBO3 -v
//
// TEST_DB_DSN が未設定の場合はスキップするため、DBの無い環境でも `make test` は通る。

const (
	bo3TestRecordId = "01JZBO3RECORD0000000000001"
	bo3TestUserId   = "bo3-test-user"
)

// setup4MatchBO3 はテスト用DBに接続し、matches/games/records を初期化して
// 対戦結果が紐づく親recordを1件作成する。
func setup4MatchBO3(t *testing.T) (repository.MatchInterface, *gorm.DB) {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN が未設定のためスキップします")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// 各テストは独立させたいので毎回クリアする
	require.NoError(t, db.Exec("DELETE FROM games").Error)
	require.NoError(t, db.Exec("DELETE FROM matches").Error)
	require.NoError(t, db.Exec("DELETE FROM records").Error)

	now := time.Now().Local()
	require.NoError(t, db.Exec(
		"INSERT INTO records (id, created_at, updated_at, user_id) VALUES (?, ?, ?, ?)",
		bo3TestRecordId, now, now, bo3TestUserId,
	).Error)

	return NewMatch(db), db
}

// newBO3Game はBO3のNゲーム目に相当するGameエンティティを組み立てる。
// created_at は「1本目 → 2本目 → 3本目」の順序が一意に定まるよう明示的にずらす。
func newBO3Game(matchId string, index int, goFirst bool, winningFlg bool) *entity.Game {
	return entity.NewGame(
		fmt.Sprintf("01JZBO3GAME%015d", index),
		time.Now().Local().Add(time.Duration(index)*time.Second),
		matchId,
		bo3TestUserId,
		goFirst,
		winningFlg,
		0,
		0,
		fmt.Sprintf("%d本目", index),
	)
}

// newBO3Match はBO3の対戦結果エンティティを組み立てる。
// victoryFlg は対戦全体の勝敗(先取2本を取れたか)を表す。
func newBO3Match(matchId string, victoryFlg bool, games []*entity.Game) *entity.Match {
	return entity.NewMatch(
		matchId,
		time.Now().Local(),
		bo3TestRecordId,
		"",
		"",
		bo3TestUserId,
		"",
		true, // bo3Flg
		false,
		false,
		false,
		false,
		false,
		victoryFlg,
		false,
		"リザードンex",
		"",
		games,
		nil,
	)
}

func TestMatchBO3(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"Create_3ゲームのBO3を作成して取得できる":         test_MatchBO3_Create3Games,
		"Create_2ゲームのBO3(2-0決着)を作成して取得できる":  test_MatchBO3_Create2Games,
		"Update_ゲーム数を3から2に減らすと2ゲームになる":      test_MatchBO3_UpdateShrink,
		"Update_ゲーム数を1から3に増やすと3ゲームになる":      test_MatchBO3_UpdateGrow,
		"Update_ゲーム内容の上書きが順序どおり反映される":       test_MatchBO3_UpdateOverwrite,
		"FindByRecordId_ゲーム数を減らした後も2ゲームになる": test_MatchBO3_FindByRecordIdAfterShrink,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// BO3で3ゲーム(1勝1敗1勝 = 2-1で勝利)を作成し、FindByIdで3ゲーム取得できること。
func test_MatchBO3_Create3Games(t *testing.T) {
	r, _ := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3MATCH00000000000001"
	games := []*entity.Game{
		newBO3Game(matchId, 1, true, true),   // 1本目: 先攻・勝ち
		newBO3Game(matchId, 2, false, false), // 2本目: 後攻・負け
		newBO3Game(matchId, 3, true, true),   // 3本目: 先攻・勝ち
	}
	require.NoError(t, r.Create(ctx, newBO3Match(matchId, true, games)))

	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)

	require.True(t, got.BO3Flg, "bo3_flg が true であること")
	require.True(t, got.VictoryFlg, "2-1で勝利しているので victory_flg は true")
	require.Len(t, got.Games, 3, "3ゲームが取得できること")

	// 1本目 → 2本目 → 3本目 の順序が保たれていること
	require.Equal(t, "1本目", got.Games[0].Memo)
	require.Equal(t, "2本目", got.Games[1].Memo)
	require.Equal(t, "3本目", got.Games[2].Memo)

	// ゲーム勝ち数と対戦全体の勝敗が整合していること(2勝1敗)
	wins := 0
	for _, g := range got.Games {
		if g.WinningFlg {
			wins++
		}
	}
	require.Equal(t, 2, wins)
	require.Equal(t, wins >= 2, got.VictoryFlg)
}

// BO3で2ゲーム(2-0決着)を作成し、FindByIdで2ゲーム取得できること。
func test_MatchBO3_Create2Games(t *testing.T) {
	r, _ := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3MATCH00000000000002"
	games := []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, true),
	}
	require.NoError(t, r.Create(ctx, newBO3Match(matchId, true, games)))

	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)

	require.True(t, got.BO3Flg)
	require.Len(t, got.Games, 2, "2-0決着なので2ゲームであること")
}

// BO3の3ゲームを「実は2-0だった」と2ゲームに修正した場合、
// 3本目は削除され、取得結果は2ゲームになること。
func test_MatchBO3_UpdateShrink(t *testing.T) {
	r, db := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3MATCH00000000000003"
	games := []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, false),
		newBO3Game(matchId, 3, true, true),
	}
	require.NoError(t, r.Create(ctx, newBO3Match(matchId, true, games)))

	// 3ゲーム → 2ゲーム(2-0)に修正する
	shrunk := []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, true),
	}
	require.NoError(t, r.Update(ctx, newBO3Match(matchId, true, shrunk)))

	// DB上、3本目のgamesは論理削除されているはず
	var deletedCount int64
	require.NoError(t, db.Raw(
		"SELECT COUNT(*) FROM games WHERE match_id = ? AND deleted_at IS NOT NULL", matchId,
	).Scan(&deletedCount).Error)
	require.Equal(t, int64(1), deletedCount, "3本目が論理削除されていること")

	// 取得結果には論理削除済みのゲームが含まれてはならない
	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)
	require.Len(t, got.Games, 2, "論理削除済みの3本目が取得結果に含まれてはならない")

	wins := 0
	for _, g := range got.Games {
		if g.WinningFlg {
			wins++
		}
	}
	require.Equal(t, 2, wins, "2-0なのでゲーム勝ち数は2であること")
}

// BO1の1ゲームをBO3の3ゲームに拡張した場合、3ゲームになること。
func test_MatchBO3_UpdateGrow(t *testing.T) {
	r, _ := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3MATCH00000000000004"
	require.NoError(t, r.Create(ctx, newBO3Match(matchId, true, []*entity.Game{
		newBO3Game(matchId, 1, true, true),
	})))

	grown := []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, false),
		newBO3Game(matchId, 3, true, true),
	}
	require.NoError(t, r.Update(ctx, newBO3Match(matchId, true, grown)))

	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)
	require.Len(t, got.Games, 3, "1ゲームから3ゲームに増えていること")
	require.Equal(t, "1本目", got.Games[0].Memo)
	require.Equal(t, "2本目", got.Games[1].Memo)
	require.Equal(t, "3本目", got.Games[2].Memo)
}

// 既存ゲームの内容(勝敗/先攻後攻)を上書きした場合、順序どおりに反映されること。
func test_MatchBO3_UpdateOverwrite(t *testing.T) {
	r, _ := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3MATCH00000000000005"
	require.NoError(t, r.Create(ctx, newBO3Match(matchId, true, []*entity.Game{
		newBO3Game(matchId, 1, true, true),   // 先攻・勝ち
		newBO3Game(matchId, 2, false, false), // 後攻・負け
		newBO3Game(matchId, 3, true, true),   // 先攻・勝ち
	})))

	// 1本目を「後攻・負け」に、3本目を「後攻・負け」に修正 → 1勝2敗で敗北
	require.NoError(t, r.Update(ctx, newBO3Match(matchId, false, []*entity.Game{
		newBO3Game(matchId, 1, false, false),
		newBO3Game(matchId, 2, false, true),
		newBO3Game(matchId, 3, false, false),
	})))

	got, err := r.FindById(ctx, matchId)
	require.NoError(t, err)
	require.Len(t, got.Games, 3)

	require.False(t, got.Games[0].WinningFlg, "1本目は負けに上書きされていること")
	require.True(t, got.Games[1].WinningFlg, "2本目は勝ちに上書きされていること")
	require.False(t, got.Games[2].WinningFlg, "3本目は負けに上書きされていること")
	require.False(t, got.VictoryFlg, "1勝2敗なので敗北")
}

// ゲーム数を減らした後、FindByRecordId でも論理削除済みゲームが含まれないこと。
func test_MatchBO3_FindByRecordIdAfterShrink(t *testing.T) {
	r, _ := setup4MatchBO3(t)
	ctx := context.Background()

	matchId := "01JZBO3MATCH00000000000006"
	require.NoError(t, r.Create(ctx, newBO3Match(matchId, true, []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, false),
		newBO3Game(matchId, 3, true, true),
	})))

	require.NoError(t, r.Update(ctx, newBO3Match(matchId, true, []*entity.Game{
		newBO3Game(matchId, 1, true, true),
		newBO3Game(matchId, 2, false, true),
	})))

	matches, err := r.FindByRecordId(ctx, bo3TestRecordId)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Len(t, matches[0].Games, 2, "論理削除済みの3本目が取得結果に含まれてはならない")
}
