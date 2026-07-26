package entity

import "testing"

func TestIsValidMatchResult(t *testing.T) {
	tests := []struct {
		name string
		in   MatchResultInput
		want bool
	}{
		// --- BO1 ---
		{"BO1_1ゲーム勝ち", MatchResultInput{VictoryFlg: true, GameWinningFlgs: []bool{true}}, true},
		{"BO1_1ゲーム負け", MatchResultInput{VictoryFlg: false, GameWinningFlgs: []bool{false}}, true},
		{"BO1_勝敗不一致", MatchResultInput{VictoryFlg: true, GameWinningFlgs: []bool{false}}, false},
		{"BO1_0ゲーム", MatchResultInput{VictoryFlg: true, GameWinningFlgs: []bool{}}, false},
		{"BO1_2ゲーム", MatchResultInput{VictoryFlg: true, GameWinningFlgs: []bool{true, true}}, false},

		// --- BO3 勝ち/負け ---
		{"BO3_2-0で勝利", MatchResultInput{BO3Flg: true, VictoryFlg: true, GameWinningFlgs: []bool{true, true}}, true},
		{"BO3_0-2で敗北", MatchResultInput{BO3Flg: true, VictoryFlg: false, GameWinningFlgs: []bool{false, false}}, true},
		{"BO3_2-1で勝利", MatchResultInput{BO3Flg: true, VictoryFlg: true, GameWinningFlgs: []bool{true, false, true}}, true},
		{"BO3_1-2で敗北", MatchResultInput{BO3Flg: true, VictoryFlg: false, GameWinningFlgs: []bool{false, true, false}}, true},
		{"BO3_2連勝なのに敗北", MatchResultInput{BO3Flg: true, VictoryFlg: false, GameWinningFlgs: []bool{true, true}}, false},
		{"BO3_2-0で決着済みなのに3ゲーム", MatchResultInput{BO3Flg: true, VictoryFlg: true, GameWinningFlgs: []bool{true, true, true}}, false},
		{"BO3_3ゲーム目の勝敗と対戦勝敗不一致", MatchResultInput{BO3Flg: true, VictoryFlg: true, GameWinningFlgs: []bool{true, false, false}}, false},
		{"BO3_1ゲーム", MatchResultInput{BO3Flg: true, VictoryFlg: true, GameWinningFlgs: []bool{true}}, false},
		{"BO3_4ゲーム", MatchResultInput{BO3Flg: true, VictoryFlg: true, GameWinningFlgs: []bool{true, false, true, true}}, false},

		// --- BO3 両者引き分け ---
		{"BO3_1勝1敗で引き分け", MatchResultInput{BO3Flg: true, DrawFlg: true, GameWinningFlgs: []bool{true, false}}, true},
		{"BO3_負け勝ちで引き分け", MatchResultInput{BO3Flg: true, DrawFlg: true, GameWinningFlgs: []bool{false, true}}, true},
		{"引き分けなのに2連勝", MatchResultInput{BO3Flg: true, DrawFlg: true, GameWinningFlgs: []bool{true, true}}, false},
		{"引き分けなのに3ゲーム", MatchResultInput{BO3Flg: true, DrawFlg: true, GameWinningFlgs: []bool{true, false, true}}, false},
		{"引き分けと勝ちが両立", MatchResultInput{BO3Flg: true, DrawFlg: true, VictoryFlg: true, GameWinningFlgs: []bool{true, false}}, false},
		{"BO1で引き分け", MatchResultInput{BO3Flg: false, DrawFlg: true, GameWinningFlgs: []bool{true, false}}, false},

		// --- 不戦勝/不戦敗 ---
		{"不戦勝はゲーム0件", MatchResultInput{DefaultVictoryFlg: true, VictoryFlg: true, GameWinningFlgs: []bool{}}, true},
		{"不戦敗はゲーム0件", MatchResultInput{DefaultDefeatFlg: true, VictoryFlg: false, GameWinningFlgs: []bool{}}, true},
		{"不戦勝なのにゲーム有り", MatchResultInput{DefaultVictoryFlg: true, VictoryFlg: true, GameWinningFlgs: []bool{true}}, false},
		{"不戦勝と不戦敗が両立", MatchResultInput{DefaultVictoryFlg: true, DefaultDefeatFlg: true, GameWinningFlgs: []bool{}}, false},
		{"不戦勝と引き分けが両立", MatchResultInput{BO3Flg: true, DefaultVictoryFlg: true, DrawFlg: true, VictoryFlg: true, GameWinningFlgs: []bool{}}, false},

		// --- チーム戦 ---
		{"チーム戦BO1はチーム勝敗を持てる", MatchResultInput{GroupMatchFlg: true, GroupMatchVictoryFlg: true, VictoryFlg: true, GameWinningFlgs: []bool{true}}, true},
		{"チーム戦でないのにチーム勝敗", MatchResultInput{GroupMatchFlg: false, GroupMatchVictoryFlg: true, VictoryFlg: true, GameWinningFlgs: []bool{true}}, false},
		{"BO3でチーム勝敗", MatchResultInput{BO3Flg: true, GroupMatchFlg: true, GroupMatchVictoryFlg: true, VictoryFlg: true, GameWinningFlgs: []bool{true, true}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMatchResult(tt.in); got != tt.want {
				t.Errorf("IsValidMatchResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
